#!/bin/bash

set -e

echo "Building for embedded target..."

# Build for embedded target as executable
# Use llgo directly instead of llgo.sh to avoid go.mod version check
llgo build -o test-verify.elf --target=esp32 .

echo "Checking exported symbols..."

# Get exported symbols
exported_symbols=$(nm -gU ./test-verify.elf | grep -E "(HelloWorld|LPSPI2_IRQHandler|SysTick_Handler|AddFunc)" | awk '{print $NF}')

echo ""
echo "Exported symbols:"
echo "$exported_symbols" | awk '{print "  " $0}'
echo ""

# Check expected symbols
expected=("HelloWorld" "LPSPI2_IRQHandler" "SysTick_Handler" "AddFunc")
missing=""

for symbol in "${expected[@]}"; do
    if ! echo "$exported_symbols" | grep -q "^$symbol$"; then
        missing="$missing $symbol"
    fi
done

if [ -n "$missing" ]; then
    echo "❌ Missing symbols:$missing"
    exit 1
fi

echo "✅ Symbol name mapping verification:"
echo "  HelloWorld           -> HelloWorld"
echo "  interruptLPSPI2      -> LPSPI2_IRQHandler"
echo "  systemTickHandler    -> SysTick_Handler"
echo "  Add                  -> AddFunc"
echo ""
echo "🎉 All export symbols verified successfully!"
echo ""

echo "Testing that non-embedded target rejects different export names..."
# Build without --target should fail with panic
if llgo build -o test-notarget.elf . 2>&1 | grep -q 'export comment has wrong name "LPSPI2_IRQHandler"'; then
    echo "✅ Correctly rejected different export name on non-embedded target"
else
    echo "❌ Should have panicked with 'export comment has wrong name' error"
    exit 1
fi
echo ""
echo "Note: Different symbol names are only supported on embedded targets."
