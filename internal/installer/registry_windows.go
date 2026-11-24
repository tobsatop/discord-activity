package installer

import "golang.org/x/sys/windows/registry"

// setRegistryDefault sets the default value of HKCU\<key>, creating the key.
func setRegistryDefault(key, value string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, key, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.SetStringValue("", value)
}

// deleteRegistryKey removes HKCU\<key> (which has no subkeys).
func deleteRegistryKey(key string) error {
	return registry.DeleteKey(registry.CURRENT_USER, key)
}
