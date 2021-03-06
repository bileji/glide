package action

import (
	"path/filepath"

	"github.com/Masterminds/glide/cache"
	"github.com/Masterminds/glide/cfg"
	"github.com/Masterminds/glide/dependency"
	"github.com/Masterminds/glide/msg"
	gpath "github.com/Masterminds/glide/path"
	"github.com/Masterminds/glide/repo"
)

// Install installs a vendor directory based on an existing Glide configuration.
func Install(installer *repo.Installer, strip, stripVendor bool) {
	if installer.UseCache {
		cache.SystemLock()
	}

	base := "."
	// Ensure GOPATH
	EnsureGopath()
	EnsureVendorDir()
	conf := EnsureConfig()

	// Lockfile exists
	if !gpath.HasLock(base) {
		msg.Info("Lock file (glide.lock) does not exist. Performing update.")
		Update(installer, false, strip, stripVendor)
		return
	}
	// Load lockfile
	lock, err := cfg.ReadLockFile(filepath.Join(base, gpath.LockFile))
	if err != nil {
		msg.Die("Could not load lockfile.")
	}
	// Verify lockfile hasn't changed
	hash, err := conf.Hash()
	if err != nil {
		msg.Die("Could not load lockfile.")
	} else if hash != lock.Hash {
		msg.Warn("Lock file may be out of date. Hash check of YAML failed. You may need to run 'update'")
	}

	// Install
	newConf, err := installer.Install(lock, conf)
	if err != nil {
		msg.Die("Failed to install: %s", err)
	}

	msg.Info("Setting references.")

	// Set reference
	if err := repo.SetReference(newConf, installer.ResolveTest); err != nil {
		msg.Err("Failed to set references: %s (Skip to cleanup)", err)
	}

	// Delete unused packages
	if installer.DeleteUnused {
		// newConf is calculated based on the lock file so it should be
		// accurate to the project list.
		dependency.DeleteUnused(newConf)
	}

	// VendoredCleanup. This should ONLY be run if UpdateVendored was specified.
	// When stripping VCS happens this will happen as well. No need for double
	// effort.
	if installer.UpdateVendored && !strip {
		repo.VendoredCleanup(newConf)
	}

	if strip {
		msg.Info("Removing version control data from vendor directory...")
		err := gpath.StripVcs()
		if err != nil {
			msg.Err("Unable to strip version control data: %s", err)
		}
	}

	if stripVendor {
		msg.Info("Removing nested vendor and Godeps/_workspace directories...")
		err := gpath.StripVendor()
		if err != nil {
			msg.Err("Unable to strip vendor directories: %s", err)
		}
	}
}
