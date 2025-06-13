# Regexy — Dangerous Function & Secrets Scanner

![banner](https://img.shields.io/badge/Regexy-Static%20Code%20Scanner-blue?style=flat-square)

Regexy is a fast, grep-powered CLI tool for scanning source code directories for dangerous functions and secrets using language-specific regular expressions.

![Regexy](https://github.com/user-attachments/assets/15344db2-1c03-4c49-961b-1acf28e980b5)
---

## Features
- Language-specific scanning using regex patterns from JSON files
- Supports `--secrets` flag to scan for hardcoded credentials, tokens, etc..
- Red highlight on the exact matched pattern
- Exclude files by extension: `--exclude .js,.css`

---

## Installation
```bash
git clone https://github.com/AbdullahAlmutawa/regexy.git
cd regexy
go mod tidy
go build -o regexy
```

---

## Usage
```bash
./regexy -L <language|all> <path> [--secrets] [--exclude .ext1,.ext2]
```

### Examples
```bash
./regexy -L php ./myproject
./regexy -L all ./src --secrets
./regexy -L all ./codebase --secrets --exclude .js,.env,.css
```

---

## Pattern Files
Place your regex patterns inside the `Reg/` folder. One JSON file per language:

```
Reg/
├── php.json
├── java.json
├── secrets.json
```

Each file contains an array of regex strings:
```json
[
  "eval\(",
  "system\(",
  "exec\("  
]
```

---
## Contributing
If you know of more dangerous functions, insecure patterns, or secrets relevant to a specific language, feel free to contribute.

To add new patterns:

- Add them to the appropriate `.json` file in the `Reg/` folder (e.g., `php.json`, `java.json`, `secrets.json`)
- Submit a Pull Request (PR)

