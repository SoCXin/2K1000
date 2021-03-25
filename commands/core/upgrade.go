// This file is part of arduino-cli.
//
// Copyright 2020 ARDUINO SA (http://www.arduino.cc/)
//
// This software is released under the GNU General Public License version 3,
// which covers the main part of arduino-cli.
// The terms of this license can be found at:
// https://www.gnu.org/licenses/gpl-3.0.en.html
//
// You can be released from the requirements of the above licenses by purchasing
// a commercial license. Buying such a license is mandatory if you want to
// modify or otherwise use the software for commercial activities involving the
// Arduino software without disclosing the source code of your own applications.
// To purchase a commercial license, send an email to license@arduino.cc.

package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/OS-Q/S04A/arduino/cores/packagemanager"
	"github.com/OS-Q/S04A/commands"
	rpc "github.com/OS-Q/S04A/rpc/commands"
)

var (
	// ErrAlreadyLatest is returned when an upgrade is not possible because
	// already at latest version.
	ErrAlreadyLatest = errors.New("platform already at latest version")
)

// PlatformUpgrade FIXMEDOC
func PlatformUpgrade(ctx context.Context, req *rpc.PlatformUpgradeReq,
	downloadCB commands.DownloadProgressCB, taskCB commands.TaskProgressCB) (*rpc.PlatformUpgradeResp, error) {

	pm := commands.GetPackageManager(req.GetInstance().GetId())
	if pm == nil {
		return nil, errors.New("invalid instance")
	}

	// Extract all PlatformReference to platforms that have updates
	ref := &packagemanager.PlatformReference{
		Package:              req.PlatformPackage,
		PlatformArchitecture: req.Architecture,
	}
	if err := upgradePlatform(pm, ref, downloadCB, taskCB, req.GetSkipPostInstall()); err != nil {
		return nil, err
	}

	if _, err := commands.Rescan(req.GetInstance().GetId()); err != nil {
		return nil, err
	}

	return &rpc.PlatformUpgradeResp{}, nil
}

func upgradePlatform(pm *packagemanager.PackageManager, platformRef *packagemanager.PlatformReference,
	downloadCB commands.DownloadProgressCB, taskCB commands.TaskProgressCB,
	skipPostInstall bool) error {
	if platformRef.PlatformVersion != nil {
		return fmt.Errorf("upgrade doesn't accept parameters with version")
	}

	// Search the latest version for all specified platforms
	platform := pm.FindPlatform(platformRef)
	if platform == nil {
		return fmt.Errorf("platform %s not found", platformRef)
	}
	installed := pm.GetInstalledPlatformRelease(platform)
	if installed == nil {
		return fmt.Errorf("platform %s is not installed", platformRef)
	}
	latest := platform.GetLatestRelease()
	if !latest.Version.GreaterThan(installed.Version) {
		return ErrAlreadyLatest
	}
	platformRef.PlatformVersion = latest.Version

	platformRelease, tools, err := pm.FindPlatformReleaseDependencies(platformRef)
	if err != nil {
		return fmt.Errorf("platform %s is not installed", platformRef)
	}
	err = installPlatform(pm, platformRelease, tools, downloadCB, taskCB, skipPostInstall)
	if err != nil {
		return err
	}

	return nil
}
