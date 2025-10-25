package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
)

const dbusCallTimeout = 1 * time.Second

type MenuPrintOptions struct {
	Separator        string
	PrintChildren    bool
	PrintParent      bool
	ParentID         int32
	ChekmarkChecked  string
	ChekmarkUnhecked string
}

func newMenuPrintOptions() MenuPrintOptions {
	return MenuPrintOptions{
		Separator:        "---",
		PrintChildren:    true,
		PrintParent:      true,
		ParentID:         -1,
		ChekmarkChecked:  "[x]",
		ChekmarkUnhecked: "[ ]",
	}
}

func dbusCall(obj dbus.BusObject, method string, args ...any) *dbus.Call {
	ctx, cancel := context.WithTimeout(context.Background(), dbusCallTimeout)
	defer cancel()
	return obj.CallWithContext(ctx, method, 0, args...)
}

func getSystrayItems(conn *dbus.Conn) ([]string, error) {
	obj := conn.Object("org.kde.StatusNotifierWatcher", "/StatusNotifierWatcher")
	var systrayItems []string

	err := dbusCall(obj, "org.freedesktop.DBus.Properties.Get", "org.kde.StatusNotifierWatcher", "RegisteredStatusNotifierItems").Store(&systrayItems)
	if err != nil {
		return nil, fmt.Errorf("failed to get systray items: %w", err)
	}

	return systrayItems, nil
}

func splitAddress(item string) (string, dbus.ObjectPath, error) {
	parts := strings.SplitN(item, "/", 2)
	if len(parts) != 2 {
		return "", dbus.ObjectPath(""), fmt.Errorf("invalid address: %s", item)
	}

	return parts[0], dbus.ObjectPath("/" + parts[1]), nil
}

func getAppId(conn *dbus.Conn, item string) (string, error) {
	address, path, err := splitAddress(item)
	obj := conn.Object(address, path)

	var title dbus.Variant
	err = dbusCall(obj, "org.freedesktop.DBus.Properties.Get", "org.kde.StatusNotifierItem", "Id").Store(&title)
	if err != nil {
		return "", fmt.Errorf("Error getting title for %s%s: %v", address, path, err)
	}

	appName, ok := title.Value().(string)
	if !ok {
		return "", fmt.Errorf("Unrecognized title type for %s", item)
	}

	return appName, nil
}

func listApps(conn *dbus.Conn) error {
	systrayItems, err := getSystrayItems(conn)
	if err != nil {
		return fmt.Errorf("Failed to retrieve systray items: %v\n", err)
	}

	for _, item := range systrayItems {
		appId, err := getAppId(conn, item)
		if err != nil {
			continue
		}
		fmt.Printf("tray|%s\t%s\n", item, appId)
	}
	return nil
}

func printMenu(conn *dbus.Conn, appAddress string, opt MenuPrintOptions) error {
	addr, path, err := splitAddress(appAddress)
	if err != nil {
		return fmt.Errorf("Failed to split address: %v\n", err)
	}
	var menu_path dbus.ObjectPath
	obj := conn.Object(addr, path)
	err = dbusCall(obj, "org.freedesktop.DBus.Properties.Get", "org.kde.StatusNotifierItem", "Menu").Store(&menu_path)
	if err != nil {
		return fmt.Errorf("Failed to get menu path: %v\n", err)
	}

	var variable dbus.Variant
	obj = conn.Object(addr, menu_path)
	err = dbusCall(obj, "com.canonical.dbusmenu.AboutToShow", 0).Store(&variable)
	if err != nil {
		return fmt.Errorf("Failed to call AboutToShow: %v\n", err)
	}

	var rawLayout rawGetLayoutResponse
	var revision uint32
	var list []string
	obj = conn.Object(addr, menu_path)
	err = dbusCall(obj, "com.canonical.dbusmenu.GetLayout", 0, -1, list).Store(&revision, &rawLayout)
	if err != nil {
		panic(err)
		// return err
	}

	// fmt.Println(rawLayout)
	layout := convertLayout(rawLayout)
	if opt.ParentID != -1 {
		printMenuItems(layout.Items, nil, appAddress, opt)
		return nil
	}

	// menuItem := findChildrenFor(layout.items, opt.ParentID)
	// printMenuItems(menuItem.Children, nil, appAddress, opt)

	return nil
}

func printMenuItems(items []MenuItem, parents []string, address string, opt MenuPrintOptions) {
	for _, item := range items {
		props := item.Properties

		if !props.Visible {
			continue
		}

		if props.Type == "separator" && opt.Separator != "" {
			label := buildMenuLabel(parents, opt.Separator)
			fmt.Printf("-\t%s\n", label)
			continue
		}

		if len(item.Children) == 0 && !props.HasLabel {
			continue
		}

		sanitizedLabel := ""
		if props.HasLabel {
			sanitizedLabel = strings.Replace(props.Label, "_", "", 1)
		}

		if props.HasLabel {
			path := buildMenuLabel(parents, sanitizedLabel)
			display := decorateLabel(path, props, opt)
			if !props.Enabled {
				display = fmt.Sprintf("<%s>", display)
			}

			if len(item.Children) > 0 {
				if opt.PrintParent {
					fmt.Printf("menu|%d|%s\t%s >\n", item.ID, address, display)
				}
			} else {
				fmt.Printf("action|%d|%s\t%s\n", item.ID, address, display)
			}
		}

		if len(item.Children) > 0 && opt.PrintChildren {
			if props.HasLabel {
				parents = append(append([]string{}, parents...), sanitizedLabel)
			}
			printMenuItems(item.Children, parents, address, opt)
			continue
		}
	}
}

func buildMenuLabel(parents []string, label string) string {
	parts := make([]string, 0, len(parents)+1)
	parts = append(parts, parents...)
	if label != "" {
		parts = append(parts, label)
	}
	return strings.Join(parts, " > ")
}

func decorateLabel(base string, props MenuProperties, opt MenuPrintOptions) string {
	if props.ToggleType == "" {
		return base
	}

	if props.ToggleType == "checkmark" {
		state := opt.ChekmarkUnhecked
		if props.ToggleState {
			state = opt.ChekmarkChecked
		}
		return fmt.Sprintf("%s %s", base, state)
	}

	state := "off"
	if props.ToggleState {
		state = "on"
	}
	return fmt.Sprintf("%s [%s:%s]", base, props.ToggleType, state)
}

func main() {
	list := false
	handle := ""

	if len(os.Args) == 1 {
		list = true
	}
	if len(os.Args) == 2 {
		handle = os.Args[1]
	}

	if !list && handle == "" {
		os.Exit(1)
	}

	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	if list {
		listApps(conn)
	}

	printOptions := newMenuPrintOptions()
	if handle != "" {
		parts := strings.Split(handle, "|")
		if len(parts) == 2 && parts[0] == "tray" {
			err := printMenu(conn, parts[1], printOptions)
			if err != nil {
				panic(err)
			}
		}
		if len(parts) == 3 && parts[0] == "menu" {
			parentId, err := strconv.Atoi(parts[1])
			if err != nil {
				panic(fmt.Errorf("invalid parent id: %s", parts[1]))
			}
			printOptions.ParentID = int32(parentId)
			err = printMenu(conn, parts[2], printOptions)

			if err != nil {
				panic(err)
			}
		}
	}
}
