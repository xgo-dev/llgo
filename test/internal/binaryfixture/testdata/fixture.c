/* Parser fixture only: these bytes are inspected, never executed. */
volatile int fixture_value = 7;

int llgo_fixture(int value) {
  return value + fixture_value;
}
