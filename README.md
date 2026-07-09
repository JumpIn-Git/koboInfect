# kobo-infect

User friendly CLI to infect Kobo eReaders (inspired by nixos-infect) with Plato, KOReader and NickelMenu. Also provides firmware updates using kobo's API. 
test-kobo/ has a fake kobo root, might be usefull to other devs.

## Usage

```
kobo-infect [-kobo <path>] [-nm-config <file>] [-sideloadMode]
```

| Flag | Description |
|---|---|
| `-kobo <path>` | Path to Kobo root (auto-detected if omitted) |
| `-nm-config <file>` | Optional NickelMenu config file to copy |
| `-sideloadMode` | Enable sideloaded mode (no account needed)  |

## Credits

- [pgaskin/koboutils](https://github.com/pgaskin/koboutils)
- [pgaskin/nickelmenu](https://github.com/pgaskin/nickelmenu) 
- [koreader/koreader](https://github.com/koreader/koreader) 
- [baskerville/plato](https://github.com/baskerville/plato)
- Other librarys used
