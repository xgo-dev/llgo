# esp-idf-audiocheck

opus/wakenet 真机适配准备固件：一块 ESP32-S31 音频板（Function-CoreBoard-1，
ES8311 codec @ I2C 0x18、MCLK=GPIO52、I2S 数据引脚、NS4150B 功放 @ GPIO57、
板载麦克风）一次烧录，串行跑完三个阶段，把 opus / 语音闭环 / wakenet 的
全部上电问题在桌面上回答掉。

```
阶段A  opus 自检（全自动）      1kHz 正弦+噪声 3s@16kHz，60/40/20ms 三种帧长
                                各编解码一轮，打印 SNR/相关系数/max 误差并自门禁
阶段B  voice-loop 一次闭环（自动） 能量 VAD 听到说话 → 预滚环 + 定长录 2s
                                → Opus 编码 → 解码 → 播放回放（打印每步字节数/耗时）
阶段C  wakenet（真机需人配合，无限循环）
                                持续监听麦克风，检出「Hi, Lexin」打
                                [go] ac C HIT n=... 并响一声功放 beep；
                                每 5s 打印一次 listening... 心跳
```

## 运行模式（启动时自动探测，同 esp-idf-voice-loop）

探测 ES8311 的 I2C 应答（0x18 读 chip ID 0x83）：

- **HW**：应答 → 应用审计过的 16kHz 配置（MCLK=256fs，MIC1 增益开满），
  真实 I2S 采集/播放 + 功放。阶段B 对着板子说话触发；阶段C 用真麦克风，
  喊「Hi Lexin」触发（真机 only）。
- **EMU**（esp-emu 无 codec）：不应答 → 阶段B 走合成场景（340+680Hz+噪声，
  与 VAD/编码/解码/对比同一管线）；阶段C 用 esp-sr 测试音频（hilexin，
  经 wakenet glue 的 test source 回放，同 esp-idf-wakenet 的 emu 形态）。
  golden `golden/emu-audiocheck.txt` 钉住这条路径。

## 阶段C 的 beep 与引脚独占（HW）

检出后的提示音不是简单的 I2S 抢占：引擎先 `wakenet.Stop()` 停车（glue 的
采集通道被 disable），beep 用 `pkg/esp/i2s` 开一条 TX 通道——IDF 的
`I2S_NUM_AUTO` 分配会落在采集通道所在的同一控制器（RX 槽被占、TX 槽空闲），
BCLK/WS/MCLK 时钟参数与采集配置完全一致（std 模式恒为 2 slot：4.096MHz
MCLK / 512kHz BCLK）——播完 240ms 后关通道，`wakenet.Start()` 恢复采集。
编解码器的时钟引脚从不会被两个 master 同时驱动；EMU 下不创建 TX 通道
（无 codec），但保留 Stop/Start 舞步，路径本身被 golden 覆盖。

## MCLK（真机麦克风的关键）

wakenet glue 模板的 I2S RX 配置不带 MCLK（`I2S_GPIO_UNUSED`）；本板的
ES8311 没有 MCLK 就不出麦克风数据。`idfproj/host/post-sync.sh` 在同步后
给模板打补丁：gpio_cfg 的 mclk 引到 demo glue 的 `llgo_audiocheck_mclk`
全局（Go 侧 HW 模式在 `wakenet.Init` 前设 52）。两处补丁点都有断言，上游
模板漂移会直接构建失败而不是静默丢 MCLK。

## 内存账 / 栈

- 栈：`-goroutine-stack-size 16384`（main fiber 64KiB）+ 128KiB GC 窗口
  （`host/artifact.conf`）——opus 深 C 调用的实测规则，照抄 voice-loop。
- opus enc/dec 状态、录音缓冲（~78KiB）、hilexin 音频（C 侧 flash const）
  的去向由 `[c] ac heap` 行打印（internal / psram 两池）。
- wakenet 引擎跑在 glue 自己的 core-1 任务上（PIE 只在 core 1 存在；
  detect 永不从 Go 侧发起，`clean()` 边沿触发永不调——pkg/esp/wakenet
  的既有约定，照抄 esp-idf-wakenet demo）。

## 构建 / emu / 烧录

```sh
# 0. 一次性：生成 wakenet_pcm.c（hilexin 音频，~60k 行数组字面量，不在 git）
cd _demo/esp-idf-audiocheck && ./gen-data.sh

# 1. 宿主侧产物（需要 llgo 与 esp-clang，见 _demo/README.md）
LLGO_ROOT=<repo> LLGO_BIN=<llgo> bash idfproj/host/make-artifact.sh <llgo>

# 2. ESP-IDF 固件（本机 IDF 无 S31 组件，走 docker；esp-sr/esp-opus 由
#    component manager 拉取，fftutil.c.obj 在 configure 时剔除）
cd idfproj
docker run --rm -v "$PWD":/project -w /project \
    espressif/idf:latest bash -c \
    'idf.py set-target esp32s31 && idf.py build && idf.py merge-bin -o merged-binary.bin'

# 3. emu 验收（单实例锁 /tmp/esp-emu.lock）
flock /tmp/esp-emu.lock ~/.local/bin/esp-emu --chip esp32s31 \
    --firmware idfproj/build/merged-binary.bin --console uart0 \
    --psram-size 16M --timeout 300s --exit-on "[go] ac done" 2>&1 | tee /tmp/audiocheck-emu.log

# 4. golden 校验（HIT/beep 行次数随 emulator 漂移，过滤后比对；完整命令见
#    golden/emu-audiocheck.txt 头部）
grep -E '^\[go\] ac' /tmp/audiocheck-emu.log | tr -d '\r' | grep -vE 'HIT n =|beep skip' \
    | sed -E 's/rms = [0-9]+ floor = [0-9]+/rms = * floor = */; s/enc-ms = [0-9]+/enc-ms = */;
              s/dec-ms = [0-9]+/dec-ms = */; s/encode ms = [0-9]+/encode ms = */;
              s/decode-play ms = [0-9]+/decode-play ms = */; s/bytes = [0-9]+/bytes = */;
              s/ratio = [0-9]+/ratio = */; s/snr = [-0-9.]+/snr = */;
              s/corr = [0-9.]+/corr = */; s/max = [0-9]+/max = */;
              s/final hits = [0-9]+ frames = [0-9]+/final hits = * frames = */;
              s/s frames = [0-9]+/s frames = */; s/hits = [0-9]+/hits = */;
              s/avg_us = [0-9]+/avg_us = */; s/max_us = [0-9]+/max_us = */' \
    | diff - <(grep -v '^#' golden/emu-audiocheck.txt) && echo GOLDEN-OK
```

真机烧录（由主控执行，本 demo 只准备固件）：

```sh
flock /tmp/s31-board.lock esptool --chip esp32s31 \
    --port /dev/cu.usbserial-110 --baud 460800 \
    write-flash 0x0 idfproj/build/merged-binary.bin
```

## 串口观察指引（真机）

- 启动行：`[go] ac mode HW (ES8311 answering at 0x18; ...)`——没打出 HW
  说明 codec 探测失败（接线/供电），后面阶段全按 EMU 跑。
- 阶段A 自动跑 ~十几秒，每个帧长一行 PASS；`enc-ms` 才是实时预算的证据
  （emu 下该数字无意义）。
- 阶段B：`[go] ac B listening` 后**对着板子说话**；说完 ~0.5s 后听到自己
  的回放即为闭环成功（`speech-cap ms = 2010 samples = 39360` 附近）。
- 阶段C：**喊「Hi Lexin」**；每次检出打 `HIT n=...` 并响一声 beep，
  每 5s 一行 `listening... t = Ns frames = ... hits = ...`。无限循环，
  断电结束（无 `[go] ac done` 行）。
- `Hit` 无 beep / 无 HIT：看 `[c] ac heap` 的 internal 余量与
  `[go] ac C error` 行。

## 分区 / flash 大小

16MB flash 布局（esp-sr S31 CI 标准）：factory 8000K + model 7900K，
wn9_hilexin 模型 284KB。构建结束 idf.py 会打印各分区占用——factory
分区用不满（本固件无 WiFi/BT，app ~1.5MB 量级）。若板子 flash <16MB，
改两处即可缩表：`post-sync.sh` 的分区 CSV（如 factory 2500K + model 512K，
模型分区 ≥284KB+余量）和 `sdkconfig.defaults` 的
`CONFIG_ESPTOOLPY_FLASHSIZE_16MB=y`→`4MB`。该缩表组合未在真板验证过。

## 真机待验清单（emu 验不了的部分）

1. 阶段A/B 的 `enc-ms` / `ratio`：真机才算实时预算（gate：B 阶段 ratio<50%）。
2. 阶段B 真麦克风 VAD 开句（底噪/增益：floor 限幅 [60,4000] 是否适配房间），
   播放回声是否被 30 帧静音窗覆盖（若自触发，加大 mute 或 vadOpenNum→3）。
3. 阶段C 真麦克风检出率（DET_MODE_95 默认阈值；响度不足先查 0x14/0x17 增益）。
4. beep：codec DAC→NS4150B 路径上的可听度（音量可调 `beepAmp`）。
5. Stop→beep→Start 舞步的引脚交接（同控制器 TX 槽复用）：若 Start 后
   frames 停涨，查 IDF i2s 日志里 beep 通道落在哪个控制器。
