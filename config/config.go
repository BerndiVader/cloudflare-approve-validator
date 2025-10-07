package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func Load(filename string, config any) error {

	if exe, err := os.Executable(); err != nil {
		return err
	} else {
		conf := filepath.Join(filepath.Dir(exe), filename)
		conf = strings.ReplaceAll(conf, "\\", "/")
		if file, err := os.Open(conf); err == nil {
			defer file.Close()
			decoder := json.NewDecoder(file)
			if err := decoder.Decode(&config); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return err
			} else {
				fmt.Fprintln(os.Stdout, "Config loaded.")
			}
		} else {
			return err
		}
	}
	return nil
}

func Save(filename string, config any) error {
	if exe, err := os.Executable(); err != nil {
		return err
	} else {
		conf := filepath.Join(filepath.Dir(exe), filename)
		conf = strings.ReplaceAll(conf, "\\", "/")

		if file, err := os.Create(conf); err == nil {
			defer file.Close()
			encoder := json.NewEncoder(file)
			encoder.SetEscapeHTML(true)
			encoder.SetIndent("", "  ")
			if err := encoder.Encode(config); err == nil {
				fmt.Fprintf(os.Stdout, "Default %s was created, add config to it.\n", filename)
			} else {
				fmt.Fprintln(os.Stderr, err)
			}
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
		fmt.Fprintln(os.Stderr, err)
	}
	return nil

}
