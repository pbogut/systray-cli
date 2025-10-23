package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/godbus/dbus/v5"
)

func getSystrayItems(conn *dbus.Conn) ([]string, error) {
	obj := conn.Object("org.kde.StatusNotifierWatcher", "/StatusNotifierWatcher")
	var systrayItems []string

	err := obj.Call("org.freedesktop.DBus.Properties.Get", 0, "org.kde.StatusNotifierWatcher", "RegisteredStatusNotifierItems").Store(&systrayItems)
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
	err = obj.Call("org.freedesktop.DBus.Properties.Get", 0, "org.kde.StatusNotifierItem", "Id").Store(&title)
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
		fmt.Println(appId)
	}
	return nil
}

func getAddressById(conn *dbus.Conn, appId string) (string, dbus.ObjectPath, error) {
	systrayItems, err := getSystrayItems(conn)
	if err != nil {
		panic(fmt.Errorf("Failed to retrieve systray items: %v\n", err))
	}

	for _, item := range systrayItems {
		tmpAppId, _ := getAppId(conn, item)
		if appId == tmpAppId {
			addr, path, err := splitAddress(item)
			if err != nil {
				return "", dbus.ObjectPath(""), err
			}
			return addr, path, nil
		}
	}
	return "", dbus.ObjectPath(""), fmt.Errorf("Application not found: %s", appId)
}

type MenuItem struct {
	ID         int32
	Properties map[string]any
	Children   []MenuItem // Adjust if children are further structured
}

type GetLayoutResponse struct {
	Version   int32
	RootProps map[string]dbus.Variant
	Items     []MenuItem
}

func printMenu(conn *dbus.Conn, appId string) error {
	addr, path, err := getAddressById(conn, appId)
	if err != nil {
		panic(err)
		// return err
	}
	var menu_path dbus.ObjectPath
	obj := conn.Object(addr, path)
	err = obj.Call("org.freedesktop.DBus.Properties.Get", 0, "org.kde.StatusNotifierItem", "Menu").Store(&menu_path)
	if err != nil {
		panic(err)
		// return err
	}
	fmt.Println(menu_path)

	var variable dbus.Variant
	obj = conn.Object(addr, menu_path)
	err = obj.Call("com.canonical.dbusmenu.AboutToShow", 0, 0).Store(&variable)
	if err != nil {
		panic(err)
		// return err
	}

	var layout GetLayoutResponse
	var revision uint32
	var list []string
	obj = conn.Object(addr, menu_path)
	err = obj.Call("com.canonical.dbusmenu.GetLayout", 0, 0, -1, list).Store(&revision, &layout)
	if err != nil {
		panic(err)
		// return err
	}
	for _, item := range layout.Items {
		enabled := true
		visible := true
		label := ""
		if item.Properties["type"] != nil {
			if item.Properties["type"].(string) == "separator" {
				fmt.Println(" \t---")
			}
		}
		if item.Properties["enabled"] != nil {
			enabled = item.Properties["enabled"].(bool)
		}
		if item.Properties["visible"] != nil {
			visible = item.Properties["visible"].(bool)
		}
		if item.Properties["type"] != nil {
		}
		if item.Properties["label"] != nil {
			label = item.Properties["label"].(string)
			label = strings.Replace(label, "_", "", 1)
			if visible {
				if enabled {
					if len(item.Children) == 0 {
						fmt.Printf("%d\t%s\n", item.ID, label)
					} else {
						fmt.Printf("%d\t%s ->\n", item.ID, label)
					}
				} else {
					fmt.Printf(" \t<%s>\n", label)
				}
			}
		}
	}

	return nil
}

func main() {
	list := flag.Bool("list", false, "List all systray items")
	menu := flag.Bool("menu", false, "Print menu items for application")
	appid := flag.String("app", "", "Application to print menu items for")
	flag.Parse()

	if !*list && !*menu {
		flag.PrintDefaults()
		os.Exit(0)
	}

	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	if *list {
		listApps(conn)
	}
	if *menu && *appid != "" {
		err := printMenu(conn, *appid)
		if err != nil {
			panic(err)
		}
	}
}
