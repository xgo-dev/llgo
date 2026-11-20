/* ESP-IDF application descriptor for the ESP32-S31 flash (XIP) target.
 *
 * The second-stage bootloader validates the app by reading the descriptor at
 * image offset 0x20 — the start of the first image segment. The linker script
 * places this array at the beginning of .data, and the image writer emits
 * .data as the first segment (its VMA sorts below the flash window), so the
 * bytes land exactly where the bootloader looks.
 */

typedef struct {
    unsigned int magic_word;
    unsigned int secure_version;
    unsigned int reserv1[2];
    char version[32];
    char project_name[32];
    char time[16];
    char date[16];
    char idf_ver[32];
    unsigned char app_elf_sha256[32];
    unsigned short min_efuse_blk_rev_full;
    unsigned short max_efuse_blk_rev_full;
    unsigned char mmu_page_size;
    unsigned char spi_flash_mode;
    unsigned char reserv3[2];
    unsigned int reserv2[18];
} esp_app_desc_t;

__attribute__((section(".appdesc"), used, aligned(4)))
const esp_app_desc_t llgo_app_desc = {
    .magic_word = 0xABCD5432u,
    .version = "1.0.0",
    .project_name = "llgo",
    .time = "00:00:00",
    .date = "Jan  1 2025",
    .idf_ver = "llgo-baremetal",
    .min_efuse_blk_rev_full = 0,
    .max_efuse_blk_rev_full = 0xFFFF,
    .mmu_page_size = 5, /* log2(32 KiB MMU pages), IDF default */
    .spi_flash_mode = 2, /* DIO, matching the image header */
};
