//go:build darwin || linux
// +build darwin linux

package fs

import (
	"encoding/json"
	"github.com/qpixel/framework"
	tlog "github.com/ubergeek77/tinylog"
	"golang.org/x/sys/unix"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"sync"
)

// fs.go
// This file contains functions that pertain to interacting with the filesystem, including mutex locking of files

var log = tlog.NewTaggedLogger("BotCore", tlog.NewColor("38;5;111"))

// GuildsDir
// The directory to use for reading and writing guild .json files. Defaults to ./guilds
// todo abstract this into a database module (being completed in feature/database)
var GuildsDir = "./guilds"

// saveLock
// A map that stores mutexes for each guild, which will be locked every time that guild's data is written
// This ensures files are written to synchronously, avoiding file race conditions
var saveLock = make(map[string]*sync.Mutex)

// loadGuilds
// Load all known guilds from the filesystem, from inside GuildsDir
func loadGuilds() (guilds map[string]*framework.Guild) {
	// Check if the configured guild directory exists, and create it if otherwise
	if _, existErr := os.Stat(GuildsDir); os.IsNotExist(existErr) {
		mkErr := os.MkdirAll(GuildsDir, 0755)
		if mkErr != nil {
			log.Fatalf("Failed to create guild directory: %s", mkErr)
		}
		log.Warningf("There are no Guilds to load; data for new Guilds will be saved to: %s", GuildsDir)

		// There are no guilds to load, so we can return early
		return guilds
	}

	// Get a list of files in the directory
	guilds = make(map[string]*framework.Guild)
	files, rdErr := ioutil.ReadDir(GuildsDir)
	if rdErr != nil {
		log.Fatalf("Failed to read guild directory: %s", rdErr)
	}

	// Iterate over each file
	for _, file := range files {
		// Ignore directories
		if file.IsDir() {
			continue
		}

		// Get the file name, convert to lowercase so ".JSON" is also valid
		fName := strings.ToLower(file.Name())

		// File name must end in .json
		if !strings.HasSuffix(fName, ".json") {
			continue
		}

		// Split ".json" from the string name, and check that the remaining characters:
		// - Add up to at least 17 characters (it must be a Discord snowflake)
		// - Are all numbers
		guildId := strings.Split(fName, ".json")[0]
		if len(guildId) < 17 || guildId != framework.EnsureNumbers(guildId) {
			continue
		}

		// Even though we are reading files, we need to make sure we can write to this file later
		fPath := path.Join(GuildsDir, fName)
		err := unix.Access(fPath, unix.O_RDWR)
		if err != nil {
			log.Errorf("File \"%s\" is not writable; guild %s WILL NOT be loaded! (%s)", fPath, guildId, err)
			continue
		}

		// Try reading the file
		jsonBytes, err := ioutil.ReadFile(fPath)
		if err != nil {
			log.Errorf("Failed to read \"%s\"; guild %s WILL NOT be loaded! (%s)", fPath, guildId, err)
			continue
		}

		// Unmarshal the json
		var gInfo framework.GuildInfo
		err = json.Unmarshal(jsonBytes, &gInfo)
		if err != nil {
			log.Errorf("Failed to unmarshal \"%s\"; guild %s WILL NOT be loaded! (%s)", fPath, guildId, err)
			continue
		}

		// Add the loaded guild to the map
		guilds[guildId] = &framework.Guild{
			ID:   guildId,
			Info: gInfo,
		}
	}

	if len(guilds) == 0 {
		log.Warningf("There are no guilds to load; data for new guilds will be saved to \"%s\"", GuildsDir)
		return guilds
	}

	// :)
	plural := ""
	if len(framework.Guilds) != 1 {
		plural = "s"
	}

	log.Infof("Loaded %d guild%s", len(guilds), plural)
	return guilds
}

// save
// Save a given guild object to .json
func save(g *framework.Guild) {
	// See if a mutex exists for this guild, and create if not
	if _, ok := saveLock[g.ID]; !ok {
		saveLock[g.ID] = &sync.Mutex{}
	}

	// Unlock writing when done
	defer saveLock[g.ID].Unlock()

	// Mark this guild as locked before saving
	saveLock[g.ID].Lock()

	// Create the output directory if it doesn't exist
	// This is a fatal error, since no other guilds would be savable if this fails
	if _, err := os.Stat(GuildsDir); os.IsNotExist(err) {
		mkErr := os.Mkdir(GuildsDir, 0755)
		if mkErr != nil {
			log.Fatalf("Failed to create guild output directory: %s", mkErr)
		}
	}

	// Convert the guild object to text
	jsonBytes, err := json.MarshalIndent(g.Info, "", "    ")
	if err != nil {
		log.Fatalf("Failed marshalling JSON data for guild %s: %s", g.ID, err)
	}

	// Write the contents to a file
	outPath := path.Join(GuildsDir, g.ID+".json")
	err = ioutil.WriteFile(outPath, jsonBytes, 0644)
	if err != nil {
		log.Fatalf("Write failed to %s: %s", outPath, err)
	}
}

// InitProvider
// Inits the filesystem provider
func InitProvider() framework.GuildProvider {
	return framework.GuildProvider{
		Save: save,
		Load: loadGuilds,
	}
}
