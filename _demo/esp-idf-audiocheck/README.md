# esp-idf-audiocheck

opus/wakenet 真机适配准备固件：一块 ESP32-S31 音频板（Function-CoreBoard-1，
ES8311 codec @ I2C 0x18、MCLK=GPIO52、I2S 数据引脚、NS4150B 功放 @ GPIO57、
板载麦克风）一次烧录，串行跑完三个阶段，把 opus / 语音闭环 / wakenet 的
全部上电问题在桌面上回答掉。

```
阶段A  opus 自检（全自动）      1kHz 正弦+噪声 3s@16kHz，60/40/20ms 三种帧长
                                各编解码一轮，打印 SNR/相关系数/max 误差并自门禁
阶段B  voice-loop 一次闭环（自动） 能量 VAD 听到说话 → 预滚环 + 录音（真机
                                上限 3s，说完静音 0.7s 即收尾）→ Opus 编码 →
                                解码 → 按采集峰值归一化后播放（打印每步字节数/耗时）
阶段C  wakenet（真机需人配合，无限循环）
                                持续监听麦克风，检出「你好小智」打
                                [go] ac C HIT n=...；真机上每次检出后关引擎、
                                录一句、Opus 往返、扬声器播回、重新武装；
                                每 5s 打印一次 listening... 心跳
```

## 运行模式（启动时自动探测，同 esp-idf-voice-loop）

探测 ES8311 的 I2C 应答（0x18 读 chip ID 0x83）：

- **HW**：应答 → 应用审计过的 16kHz 配置（MCLK=256fs，MIC1 增益开满），
  真实 I2S 采集/播放 + 功放。阶段B 对着板子说话触发；阶段C 用真麦克风，
  喊「你好小智」触发（真机 only）。HW 日志每行带 `+N ms` 时间戳（emu 不带，
  golden 逐行比对）。
- **EMU**（esp-emu 无 codec）：不应答 → 阶段B 走合成场景（340+680Hz+噪声，
  与 VAD/编码/解码/对比同一管线）；阶段C 用 esp-sr 测试音频（hilexin，
  经 wakenet glue 的 test source 回放，同 esp-idf-wakenet 的 emu 形态）。
  golden `golden/emu-audiocheck.txt` 钉住这条路径。

## 阶段C 的引脚独占：为什么没有提示音（HW）

检出后不能开 TX 通道做提示音。实测：引擎 `wakenet.Stop()` 停车后（glue 的
采集通道已 disable）再用 `pkg/esp/i2s` 开一条 TX 通道，`I2S_NUM_AUTO` 会把
它分到采集通道所在的同一控制器；此时 Write 一律超时（`esp error 263`，
去掉 MCLK 也一样），并且**此后采集读到的是一段恒定值**——`chunk-rms` 从
波动的 157-311 冻结在 6160，引擎再也检不出。也就是说提示音会把采集打断。
所以真机路径改成：检出 → 打日志 → 重新武装；要提示音得等引擎**完整关闭**
（`wakenet.Close()`，代价是重新加载模型），见下一条。

## 阶段C 的检出后回声（HW）

`echoAfterWake` 是助手骨架：`Stop` → `Close`（把 I2S 控制器整段让出来，
仅停车不足以让第二个通道工作）→ 复用阶段B 的采集/编码/回放循环 → `Init`
→ `Start` 重新武装。采集上限 3 s、静音 700 ms 提前收尾：VAD 通常在人开口
后才开句，2 s 固定窗会把整句的尾巴切掉（实测 2.08 s 采+播、后半句没声）。
注意两个缓冲都按 3 s 上限分配（`utter` 与 `pktSizes`）：3 s 是 153 个 20ms
包，比"3/0.02 = 150"多——按帧长整除算会差一个包，实测先在硬件上炸过。

## 播放电平（HW）

麦克风链路的电平极低（说话时 rms 22-40、峰值 130-190 ≈ 0.4% 满量程），
解码后按原样播基本听不见。阶段B/回声的播放先算采集峰值，归一到 1/4 满量程
（增益上限 32×）再送 I2S，日志打印 `play gain = N x (capture peak = P)`。
I2S TX 链路本身是好的：单独开 TX 播 1kHz（16k 与 44.1k 两种采样率）都清晰
可听，寄存器回读 `0x00=0x80 / 0x12=0x00 / 0x13=0x10 / 0x31=0x00 / 0x32=0xBF`
全部正确。

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
# 0. 一次性：生成 wakenet_pcm.c（esp-sr 的 hilexin 测试音频，~60k 行数组字面量，
#    不在 git。emu 阶段C 注入的就是它——所以 emu 的命中数与模型是否配对无关，
#    只 gate 帧数；真机才是检出门槛）
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
- 阶段B：`[go] ac B listening` 后**对着板子说话**；说完约 1-2s 后听到自己
  的回放即为闭环成功。真机实测：`speech-open` 后 `speech-cap ms = 3000
  samples = 48960`，`encode ms ≈ 1960`（64% 实时），`play gain = 3 x`，
  `decode-play ms ≈ 3056`。
- 阶段C：**喊「你好小智」**；检出打 `HIT n=...`，随后自动关引擎、`echo start
  (say a sentence)`、录一句、播回、`rearmed`。之后每 5s 一行
  `listening... t = Ns frames = ... hits = ... chunk-rms = ...`。无限循环，
  断电结束（无 `[go] ac done` 行）。
- 没有 HIT：`chunk-rms` 应随说话波动（安静房间 130-330）。若它恒定不变，
  说明采集被别的通道打断过（见上面两节），重新上电即可。

## 分区 / flash 大小

16MB flash 布局（esp-sr S31 CI 标准）：factory 8000K + model 7900K，
wn9_nihaoxiaozhi_tts 模型 284KB。构建结束 idf.py 会打印各分区占用——factory
分区用不满（本固件无 WiFi/BT，app ~1.5MB 量级）。若板子 flash <16MB，
改两处即可缩表：`post-sync.sh` 的分区 CSV（如 factory 2500K + model 512K，
模型分区 ≥284KB+余量）和 `sdkconfig.defaults` 的
`CONFIG_ESPTOOLPY_FLASHSIZE_16MB=y`→`4MB`。该缩表组合未在真板验证过。

## 真机结论（ESP32-S31-Function-CoreBoard-1）

三条链路都已在这块板上跑通，日志与数字见上面各节：

- **阶段A**：SNR 18.0/18.2/17.0 dB（60/40/20ms），相关系数 0.99；3 秒音频
  编码 1.88-1.95 秒（62% 实时），解码 106-125 ms。
- **阶段B**：VAD 开句正常，3 秒采集 → 编码 1.96 秒（64%）→ 归一化播放。
- **阶段C**：`wn9_nihaoxiaozhi_tts` 阈值 0.50 下真机检出，帧率 31.25/s
  （512 样本 @16kHz），统计 `avg_us ≈ 1.5-3 ms/chunk`。

emu 路径的差异都在这里写清楚：emu 注入 esp-sr 的 hilexin 录音，而 emu 从同一
镜像映射出来的模型是「嗨，乐鑫」（真机加载同一镜像得到的是「你好小智」——模拟
器的 srmodels 映射与真机不一致，属模拟器保真度边界）。因此 emu 的命中数只作
信息打印，检测门槛绑在真机；emu 也不播提示音（没有 codec）。

## 仍未解决 / 已知限制

1. 麦克风电平很低（说话 rms 22-40，峰值 ≈0.4% 满量程）。检出不依赖它（glue
   有 8× 软件增益），但回声必须靠播放侧归一化才听得清——根因在 codec 模拟
   前端，`0x16` 数字增益实测效果远小于其标称值。
2. 没有唤醒提示音（原因见"引脚独占"节）。要做提示音需要给引擎与播放排期
   一个共享的 I2S 使用者，或换一块能把提示音交给别处（如 LED/PWM）的板子。
3. 每次检出后回声都要 `wakenet.Close()` + `Init()` 重新加载模型（约 1 秒），
   因为采集通道的存在会让第二个 I2S 通道拿不到时钟。真正的助手应当让引擎
   与录音共用一个采集通道（引擎侧边检测边存包），这是下一步的结构性工作。
