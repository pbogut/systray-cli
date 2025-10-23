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

func main() {
	list := flag.Bool("list", false, "List all systray items")
	flag.Parse()

	if !*list {
		flag.PrintDefaults()
		os.Exit(0)
	}

	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	systrayItems, err := getSystrayItems(conn)
	if err != nil {
		panic(fmt.Errorf("Failed to retrieve systray items: %v\n", err))
	}

	for _, item := range systrayItems {
		appId, err := getAppId(conn, item)
		if err != nil {
			continue
		}
		fmt.Println(appId)
	}
}
