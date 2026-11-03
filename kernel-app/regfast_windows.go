package main

import (
	"strings"

	"golang.org/x/sys/windows/registry"
)

// Нативное чтение/запись реестра — в тысячи раз быстрее запуска reg.exe/PowerShell.

func regRoot(root string) registry.Key {
	switch strings.ToUpper(root) {
	case "HKLM":
		return registry.LOCAL_MACHINE
	case "HKCU":
		return registry.CURRENT_USER
	case "HKCR":
		return registry.CLASSES_ROOT
	case "HKU":
		return registry.USERS
	}
	return registry.CURRENT_USER
}

// Читает DWORD; ok=false, если значения нет
func regReadDword(root, path, name string) (uint32, bool) {
	k, err := registry.OpenKey(regRoot(root), path, registry.QUERY_VALUE|registry.WOW64_64KEY)
	if err != nil {
		return 0, false
	}
	defer k.Close()
	v, _, err := k.GetIntegerValue(name)
	if err != nil {
		return 0, false
	}
	return uint32(v), true
}

func regReadString(root, path, name string) (string, bool) {
	k, err := registry.OpenKey(regRoot(root), path, registry.QUERY_VALUE|registry.WOW64_64KEY)
	if err != nil {
		return "", false
	}
	defer k.Close()
	s, _, err := k.GetStringValue(name)
	if err != nil {
		return "", false
	}
	return s, true
}

func regKeyExists(root, path string) bool {
	k, err := registry.OpenKey(regRoot(root), path, registry.QUERY_VALUE|registry.WOW64_64KEY)
	if err != nil {
		return false
	}
	k.Close()
	return true
}

func regSubKeys(root, path string) []string {
	k, err := registry.OpenKey(regRoot(root), path, registry.ENUMERATE_SUB_KEYS|registry.WOW64_64KEY)
	if err != nil {
		return nil
	}
	defer k.Close()
	names, _ := k.ReadSubKeyNames(-1)
	return names
}

func regWriteDwordNative(root, path, name string, val uint32) error {
	k, _, err := registry.CreateKey(regRoot(root), path, registry.SET_VALUE|registry.WOW64_64KEY)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.SetDWordValue(name, val)
}

func regDeleteValueNative(root, path, name string) {
	k, err := registry.OpenKey(regRoot(root), path, registry.SET_VALUE|registry.WOW64_64KEY)
	if err != nil {
		return
	}
	defer k.Close()
	k.DeleteValue(name)
}

// StartType службы из реестра: 2=авто, 3=вручную, 4=отключена
func serviceStart(name string) int {
	v, ok := regReadDword("HKLM", `SYSTEM\CurrentControlSet\Services\`+name, "Start")
	if !ok {
		return -1
	}
	return int(v)
}
