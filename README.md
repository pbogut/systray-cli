# Systray :heart: CLI

A tiny command-line companion for poking at DBus StatusNotifier (system tray) items. List the running tray apps, inspect their menus, and trigger actions straight from the shell.

## Installation

Using Go tooling (requires Go 1.25+):

```sh
go install github.com/pbogut/systray-cli@latest
```

Building manually:

```sh
git clone https://github.com/pbogut/systray-cli.git
cd systray-cli
go build
```

The binary is named `systray-cli`.

## Basic Usage

- Run `systray-cli` with no arguments to print all registered tray items.
- The list output uses the format `tray|<dbus-address>\t<display-name>`.
- Pass one of those handles back to explore the menu: `systray-cli "tray|<dbus-address>"`.
- Menu entries render as `menu|<id>|<address>` when they have children and `action|<id>|<address>` when they are actionable leaf items.
- Execute an action directly with `systray-cli "action|<id>|<address>"` or list sub-menu with `systray-cli "menu|<id>|<address>"`.

`systray-cli` talks to the session bus, so it must run inside a graphical session that exposes StatusNotifier items (KDE Plasma, GNOME with extensions, Waybar, etc.).

## Configuration

`systray-cli` looks for a TOML configuration file at the XDG path `~/.config/systray/config.toml`. If the file is missing, sane defaults are used.

Supported keys:

- `separator` (string): Overrides the text printed for DBus "separator" menu items. Empty string hides separators.
- `names` (table of key/value): Maps original application IDs to friendlier display names.
- `show_parent` (bool): When true, parent menu entries preceding their children are printed.
- `show_children` (bool): Controls whether nested menu items are expanded automatically.
- `menu_indicator` (string): The text printed after the menu item name when it has children.
- `menu_separator` (string): The text printed between nested menu items.
- `checkmark_checked` (string): The text printed for checked checkmark toggles.
- `checkmark_unchecked` (string): The text printed for unchecked checkmark toggles.

Example:

```toml
separator = " — "
show_parent = true
show_children = true
menu_indicator = ""
menu_separator = "  "

[names]
"KDE Connect Indicator" = "󰄡 KDE Connect"
"nm-applet" = " Network"
```

A richer sample with glyphs lives in `examples/config.toml`.

## Tips & Tricks

- Combine `systray-cli` with fuzzy finders (`fzf`, `fuzzle`, `rofi`, etc.) to quickly drill into menu entries.
- Aliasing tray names via `names` makes the output easier to scan in busy environments.

## Contribution

PRs and issue reports are always welcome. Please keep changes scoped and include `go build` output when reporting problems.

## License

MIT License; the software is provided "as is", without warranty of any kind.
