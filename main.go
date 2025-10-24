package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
)

const dbusCallTimeout = 1 * time.Second

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
		fmt.Printf("%s\t%s\n", item, appId)
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

func printMenu(conn *dbus.Conn, appId string) error {
	addr, path, err := getAddressById(conn, appId)
	if err != nil {
		panic(err)
		// return err
	}
	var menu_path dbus.ObjectPath
	obj := conn.Object(addr, path)
	err = dbusCall(obj, "org.freedesktop.DBus.Properties.Get", "org.kde.StatusNotifierItem", "Menu").Store(&menu_path)
	if err != nil {
		panic(err)
		// return err
	}
	fmt.Println(menu_path)

	var variable dbus.Variant
	obj = conn.Object(addr, menu_path)
	err = dbusCall(obj, "com.canonical.dbusmenu.AboutToShow", 0).Store(&variable)
	if err != nil {
		panic(err)
		// return err
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

	layout := convertLayout(rawLayout)
	for _, item := range layout.Items {
		props := item.Properties

		if props.Type == "separator" {
			fmt.Println(" \t---")
			continue
		}

		if !props.HasLabel {
			continue
		}

		label := strings.Replace(props.Label, "_", "", 1)

		if !props.Visible {
			continue
		}
		if !props.Enabled {
			fmt.Printf(" \t<%s>\n", label)
			continue
		}

		if len(item.Children) == 0 {
			fmt.Printf("%d\t%s\n", item.ID, label)
		} else {
			fmt.Printf("%d\t%s ->\n", item.ID, label)
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
