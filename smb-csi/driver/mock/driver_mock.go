package mock

import (
	"os"
	"path"
	"path/filepath"
	"smb-csi/driver"
	"smb-csi/driver/state"
)

const (
	driverName    = "test"
	driverVersion = "1.0.0"
	driverStateDir	  = "./csi-data-dir-test"
)

var vol1 = state.Volume{
	VolID: "testID",
	VolName: "testName",
	VolSize: 1024,
	VolPath: filepath.Join(driverStateDir, "testID"),
}

var vol2 = state.Volume{
	VolID: "testID2",
	VolName: "testName2",
	VolSize: 2048,
	VolPath: filepath.Join(driverStateDir, "testID2"),
}

var vol3 = state.Volume{
	VolID: "testID3",
	VolName: "testName3",
	VolSize: 4096,
	VolPath: filepath.Join(driverStateDir, "testID3"),
}

var snap1 = state.Snapshot{
	Id: "snapID",
	Name: "snapName",
	VolID: "testID",
}

var snap2 = state.Snapshot{
	Id: "snapID2",
	Name: "snapName2",
	VolID: "testID2",
}

var snap3 = state.Snapshot{
	Id: "snapID3",
	Name: "snapName3",
	VolID: "testID3",
}

func NewMockDriver(nodeID string) (*driver.Driver, error) {

	if stateDirErr := os.MkdirAll(driverStateDir, 0750); os.IsExist(stateDirErr) {
		return nil, stateDirErr
	}

	smbState, stateErr := state.New(path.Join(driverStateDir, "state.json"))
	if stateErr != nil {
		return nil, stateErr
	}

	smbState.UpdateVolume(vol1)
	smbState.UpdateVolume(vol2)
	smbState.UpdateVolume(vol3)

	smbState.UpdateSnapshot(snap1)
	smbState.UpdateSnapshot(snap2)
	smbState.UpdateSnapshot(snap3)


	return &driver.Driver{
		Name:     driverName,
		Version:  driverVersion,
		StateDir: driverStateDir,
		Mounter:  *NewFakeMounter(),
		NodeID:   nodeID,
		State:    smbState,
	}, nil
}
