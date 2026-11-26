# esp-idf-voice-loop

「录音 → Opus 编码 → 解码 → 扬声器播放」的单工回声 demo：麦克风听人说话，
说完后（VAD 判定）整句经 `pkg/esp/opus` 编码再解码，从扬声器播回来。

```
待机(VAD 监听, 1s 预滚环) → 检测到说话 → 录音缓存(PSRAM, 上限 10s)
  → 静音判定(连续 17 帧 ~510ms) → Opus 编码(16kHz mono voip, 打印耗时)
  → 立即解码 → I2S TX + 功放播放 → 回待机
```

单 goroutine 顺序状态机，无并发：各阶段严格串行，每个 I2S 阶段独占主模式
时钟引脚（录音通道在播放通道打开之前关闭 —— 引脚从不同时被两个 master 驱
动；这个形状与 esp-idf-all 的音频自测在板上验证过的一致）。

## 运行模式（自动探测）

固件启动时探测 ES8311 的 I2C 应答（0x18 读 chip ID 0x83）：

- **HW**：应答 → 应用审计过的 16kHz 配置（MCLK=256fs，MIC1 增益开满），
  真实 I2S RX 采集 + TX 播放，功放 GPIO57。
- **EMU**（esp-emu 无 codec）：不应答 → 合成场景走**同一条**
  VAD → 编码 → 解码 → 对比路径，emu 验证状态机/VAD 判定/编码耗时/质量门，
  只有模拟引脚是虚拟的。golden `golden/emu-voice-loop.txt` 钉住这层。

## VAD（demo_glue.c 内整数能量 VAD，参数全部 Go 侧可调）

30ms 帧（480 采样@16kHz）RMS + 自适应噪声地板（慢速 IIR 1/64，限幅
[60, 4000]）+ 迟滞 + 尾切：

| 参数 | 默认 | 含义 |
|---|---|---|
| open 阈值 | 2×floor（≈6dB） | rms ≥ 2×地板 判语音帧 |
| open 帧数 | 3（~90ms） | 连续 3 语音帧开句；开句前 1s 预滚环内容并入录音（不吃字头） |
| hangover | 17 帧（~510ms） | 连续 17 静音帧（rms ≤ floor）关句；尾切全部留在录音里 |
| floor 限幅 | 60..4000 | 防死寂房间灰尘触发 / 防嘈杂房间永不触发 |

HW 模式额外静音窗：每个 cycle 开头 30 帧（~900ms）聋（吃掉 codec 上电
pop 和上一次播放的房间余音），采集照常排水保持 DMA 与地板健康。

## 内存账（emu 实测）

| 项 | 大小 | 去向 |
|---|---|---|
| 录音缓冲 10s mono | 320 KiB | Go 堆 → tinygogc PSRAM 段 |
| 编码包 arena | 128 KiB（上限；24kbps 实测 ~6.2KB/3.2s） | Go 堆 |
| 预滚环 1s | 31 KiB | Go 堆 |
| opus enc/dec 状态 | 24,716 / 18,136 B | malloc wrap → GC 堆（PSRAM 段） |
| `[c] vloop heap armed` 行 | boot→armed Δpsram | 见 emu 输出，PSRAM 承接全部大对象 |

栈：`-goroutine-stack-size 16384`（main fiber 64KiB）+ 128KiB GC 窗口
（`host/artifact.conf`）——opus 深 C 调用的实测栈规则见 pkg/esp/opus 文档。

## 板卡接线（与 esp-idf-all 相同，已真机验证过播放链路）

| 信号 | 引脚 |
|---|---|
| I2C SDA/SCL（ES8311 @0x18） | 51 / 50 |
| I2S MCLK/BCLK/WS | 52 / 53 / 55 |
| I2S DOUT（播放）/ DIN（录音） | 56 / 54 |
| NS4150B 功放使能 | 57（高有效） |

## 构建 / 运行

```sh
# 宿主侧产物（需要 llgo 与 LLVM，见 _demo/README.md）
cd _demo/esp-idf-voice-loop
LLGO_ROOT=<repo> LLGO_BIN=<llgo> bash idfproj/host/make-artifact.sh <llgo>
# ESP-IDF 侧
cd idfproj && idf.py set-target esp32s31 && idf.py build && idf.py merge-bin -o merged-binary.bin
# emu 验收
~/.local/bin/esp-emu --chip esp32s31 --firmware build/merged-binary.bin \
  --console uart0 --psram-size 16M --exit-on "[go] vloop done"
```

## 真机待验清单（emu 验不了的部分）

1. ES8311 录音路径实际增益（`micSweep` 的四个配置本 demo 未扫，固定
   MIC1+max PGA；若底噪大/声音小，先用 esp-idf-all 的自测确认 winner 配置）。
2. VAD 对真实语音的开/关句表现（噪声地板 60..4000 限幅是否适配实际房间）。
3. 播放回声是否被静音窗覆盖（若Speaker→Mic 声学耦合强导致自触发，加大
   `muteM` 或降 `vadOpenNum`→3）。
4. 编码实时性：`ratio` 行 <50% 为过（emu 下该数字无意义）。
5. 听感：说完到出声的延迟 ≈ hangover 510ms + 编码 + 播放实时流，预期 <1s。
