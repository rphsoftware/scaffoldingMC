package main

import (
	"encoding/json"
	"io/ioutil"
	"sort"
	"strconv"
)

type Registry struct {
	blockStates map[string]int

	blockStateFields map[string]map[string]bool

	reverseBlockStates map[int]string
}

var registry map[int]*Registry

func makeBlockStateIdentifier(name string, properties []string) string {
	output := name + ";"
	sort.Strings(properties)

	for _, i := range properties {
		output = output + i + ";"
	}

	return output
}

func loadRegistry() {
	log("Loading registry...")
	registry = make(map[int]*Registry)

	type jsonRegistryState struct {
		Properties map[string]string `json:"properties"`
		Id         int               `json:"id"`
		Default    bool              `json:"default"`
	}
	type jsonRegistryEntry struct {
		Properties map[string][]string `json:"properties"`
		States     []jsonRegistryState `json:"states"`
	}
	type jsonRegistryBase map[string]jsonRegistryEntry

	handledVersions, _ := ioutil.ReadDir("registries")
	for _, val := range handledVersions {
		log("Protocol: ", val.Name())
		var bruh jsonRegistryBase = make(map[string]jsonRegistryEntry)
		registriesInfo, _ := ioutil.ReadFile("registries/" + val.Name() + "/blocks.json")
		json.Unmarshal(registriesInfo, &bruh)
		protId, _ := strconv.Atoi(val.Name())

		registry[protId] = new(Registry)
		registry[protId].blockStateFields = make(map[string]map[string]bool)

		registry[protId].blockStates = make(map[string]int)
		registry[protId].reverseBlockStates = make(map[int]string)

		for key, value := range bruh {
			registry[protId].blockStateFields[key] = make(map[string]bool)
			//fmt.Println(key)
			for property := range value.Properties {
				registry[protId].blockStateFields[key][property] = true
			}

			// Assemble block state string
			for _, data := range value.States {
				states := []string{}
				for stateName, stateValue := range data.Properties {
					states = append(states, stateName+"="+stateValue)
				}

				fullStateName := makeBlockStateIdentifier(key, states)
				registry[protId].blockStates[fullStateName] = data.Id
				registry[protId].reverseBlockStates[data.Id] = fullStateName
			}
		}
	}

	log("Registry finished loading")
}

