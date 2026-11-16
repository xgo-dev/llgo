import unittest

from check_doc_structure import check_readme


class ReadmeAudienceTests(unittest.TestCase):
    def check(self, content):
        return check_readme(content + "\n[Contributing](CONTRIBUTING.md)\n")

    def test_user_installation_and_usage(self):
        self.assertEqual(self.check("""## How to install
```sh
brew install llvm@22 lld@22
curl -fsSL https://raw.githubusercontent.com/xgo-dev/llgo/main/install.sh | bash
sudo apt-get install -y llgo
llgo run .
llgo test ./...
```
```powershell
irm https://raw.githubusercontent.com/xgo-dev/llgo/main/install.ps1 | iex
```
"""), [])

    def test_development_commands(self):
        for command in (
            "mise -C dev en", "pixi shell --locked", "./install.sh",
            "bash install.sh", "source dev/llgo.sh",
            "go build ./cmd/llgo", "go test ./cl", "./dev/llgo.sh test ./test",
            "git clone https://github.com/xgo-dev/llgo.git",
        ):
            with self.subTest(command=command):
                self.assertTrue(self.check(f"```sh\n{command}\n```"))

    def test_tilde_fences_and_indented_commands(self):
        self.assertTrue(self.check("~~~sh\n  pixi shell --locked\n~~~"))

    def test_source_setup_headings(self):
        for heading in (
            "### Install from source", "### Manual installation",
            "## Development environment", "### with mise or Pixi",
        ):
            with self.subTest(heading=heading):
                self.assertTrue(self.check(heading))

    def test_prose_can_link_to_development_setup(self):
        self.assertEqual(self.check(
            "For mise or Pixi, see [development setup](CONTRIBUTING.md)."
        ), [])

    def test_contributor_link_required(self):
        self.assertTrue(check_readme("# LLGo\n"))


if __name__ == "__main__":
    unittest.main()
