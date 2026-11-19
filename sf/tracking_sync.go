package sf

import (
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ForceCLI/force/lib"
)

// tracksSource reports whether deploys to and retrieves from the org update
// source tracking.  The authorization file records the answer.  Without
// one, a scratch org tracks source, a sandbox does when it has the
// SourceMember object, and another org does not; the answer is then
// recorded, as sf records it.
func tracksSource(org *targetOrg) bool {
	if value, ok := org.Fields["tracksSource"].(bool); ok {
		return value
	}
	tracks := false
	switch {
	case org.Fields.Bool("isScratch") || org.Fields.String("devHubUsername") != "":
		tracks = true
	case org.Fields.Bool("isSandbox"):
		tracks, _ = supportsSourceTracking(org)
	}
	lib.UpdateSFDXAuthFields(org.Username, lib.SFDXAuthFields{"tracksSource": tracks})
	return tracks
}

// trackingPollInterval is the time between queries for the source members
// of a deploy.
var trackingPollInterval = time.Second

// componentKeys returns the keys of the components a source set deploys.
func (s *sourceSet) componentKeys() []string {
	var keys []string
	for metadataType, members := range s.types {
		for _, member := range members {
			keys = append(keys, metadataType+"###"+member)
		}
	}
	sort.Strings(keys)
	return keys
}

// conflictRows lists the files of the components in keys that changed both
// locally and in the org: the -meta.xml file of each component first.
func (s *trackingState) conflictRows(root string, keys []string) []*orderedMap {
	conflicts := s.conflicts()
	rows := []*orderedMap{}
	seen := map[string]bool{}
	for _, key := range keys {
		component := s.components[key]
		if component == nil || !conflicts[key] || seen[key] {
			continue
		}
		seen[key] = true
		files := []string{}
		if component.xml != "" {
			files = append(files, component.xml)
		}
		for _, file := range component.files {
			if file != component.xml && s.onDisk[file] {
				files = append(files, file)
			}
		}
		for _, file := range files {
			row := newOrderedMap()
			row.Set("state", "Conflict")
			row.Set("fullName", component.FullName)
			row.Set("type", component.Type)
			row.Set("filePath", filepath.Join(root, filepath.FromSlash(file)))
			rows = append(rows, row)
		}
	}
	return rows
}

// conflictError prints the table of conflicts and returns the error sf
// reports for a deploy or a retrieve that would overwrite changes.
func conflictError(cmd *cobra.Command, rows []*orderedMap, deploy bool) error {
	if !jsonRequested(cmd) {
		table := make([][]string, 0, len(rows))
		for _, row := range rows {
			table = append(table, []string{text(row.values["state"]), text(row.values["fullName"]), text(row.values["type"]), text(row.values["filePath"])})
		}
		printTable(cmd.OutOrStdout(), "", []string{"STATE", "FULL NAME", "TYPE", "FILE PATH"}, table)
		message := "There are changes in your local files that conflict with the org changes you're trying to retrieve."
		actions := []string{
			"To overwrite the local changes, rerun this command with the --ignore-conflicts flag.",
			"To overwrite the remote changes, run the \"sf project deploy start\" command with the --ignore-conflicts flag.",
		}
		if deploy {
			message = "There are changes in the org that conflict with the local changes you're trying to deploy."
			actions = []string{
				"To overwrite the remote changes, rerun this command with the --ignore-conflicts flag.",
				"To overwrite the local changes, run the \"sf project retrieve start\" command with the --ignore-conflicts flag.",
			}
		}
		return &Error{Name: "SourceConflictError", Label: "1", Message: message, ExitCode: 1, Actions: actions}
	}
	return &Error{Name: "SourceConflictError", Message: fmt.Sprintf("%d conflicts detected", len(rows)), ExitCode: 1, Data: rows}
}

// localChangeSet is what a deploy without named components deploys: the
// components with added or modified files, and the components whose files
// were all deleted.
type localChangeSet struct {
	deploy, remove []*sourceComponent
}

// localChangeSet returns the components to deploy and to delete, leaving
// out the files .forceignore excludes.
func (s *trackingState) localChangeSet(root string, ignore *forceIgnore) *localChangeSet {
	set := &localChangeSet{}
	seen := map[string]bool{}
	for _, file := range s.changes.deleted {
		if component := s.componentOf(file); component != nil && !s.exists(component) && !seen[component.key()] {
			seen[component.key()] = true
			set.remove = append(set.remove, component)
		}
	}
	changed := append(append(append([]string{}, s.changes.added...), s.changes.modified...), s.changes.deleted...)
	sort.Strings(changed)
	for _, file := range changed {
		component := s.componentOf(file)
		if component == nil || seen[component.key()] {
			continue
		}
		if s.onDisk[file] && ignore.ignores(filepath.Join(root, filepath.FromSlash(file))) {
			continue
		}
		if !s.exists(component) {
			continue
		}
		seen[component.key()] = true
		set.deploy = append(set.deploy, component)
	}
	return set
}

// metadataName returns the name by which a component is deployed.  A field
// or another part of a decomposed custom object is deployed with its
// object.
func (c *sourceComponent) metadataName() string {
	for _, childType := range objectChildTypes {
		if c.Type == childType {
			object, _, _ := strings.Cut(c.FullName, ".")
			return "CustomObject:" + object
		}
	}
	return c.Type + ":" + c.FullName
}

// destructiveChanges returns the manifest that deletes components.
func destructiveChanges(components []*sourceComponent, version string) []byte {
	types := map[string][]string{}
	for _, component := range components {
		types[component.Type] = append(types[component.Type], component.FullName)
	}
	return packageXML(types, version)
}

// trackingProgress is the outcome of the source tracking update of a
// deploy, for the progress display.
type trackingProgress struct {
	found, total int
	elapsed      time.Duration
}

// recordDeploy records a successful deploy: the files of the deployed
// components are in sync, the files of the deleted components are gone,
// and the revision the org gave each component is the one last retrieved.
// It waits for the org to report the source members of the components, for
// the time sf waits.
func (t *sourceTracking) recordDeploy(state *trackingState, deployed, deleted []string) (*trackingProgress, error) {
	started := time.Now()
	repo, err := openShadowRepo(t.gitDir(), t.project.Root)
	if err != nil {
		return nil, err
	}
	var updated, removed []string
	for _, key := range deployed {
		if component := state.components[key]; component != nil {
			for _, file := range component.files {
				if state.onDisk[file] {
					updated = append(updated, file)
				} else {
					removed = append(removed, file)
				}
			}
		}
	}
	for _, key := range deleted {
		if component := state.components[key]; component != nil {
			removed = append(removed, component.files...)
		}
	}
	if err := t.commitLocal(repo, updated, removed, "sfdx source tracking"); err != nil {
		return nil, err
	}
	keys := append(append([]string{}, deployed...), deleted...)
	found, err := t.syncRemote(keys, true)
	if err != nil {
		return nil, err
	}
	return &trackingProgress{found: found, total: len(keys), elapsed: time.Since(started)}, nil
}

// syncRemote asks the org for the source members changed since the last
// query, records them, and marks the members in keys as retrieved at their
// current revision.  With wait, it repeats the query until the org has
// reported every member in keys or the time sf allows has passed.  It
// returns the number of members in keys that the org reported.
func (t *sourceTracking) syncRemote(keys []string, wait bool) (int, error) {
	tracking, err := loadRemoteTracking(t.revisionPath())
	if err != nil {
		return 0, err
	}
	outstanding := map[string]bool{}
	for _, key := range keys {
		outstanding[key] = true
	}
	deadline := time.Now().Add(time.Duration(math.Ceil(float64(len(keys))*0.05)+5) * time.Second)
	for {
		members, err := querySourceMembers(t.org, fmt.Sprintf("RevisionCounter > %d", tracking.maxRevision))
		if err != nil {
			return 0, err
		}
		for _, member := range members {
			delete(outstanding, member.key())
		}
		if err := tracking.track(members, false); err != nil {
			return 0, err
		}
		if !wait || len(outstanding) == 0 || time.Now().After(deadline) {
			break
		}
		time.Sleep(trackingPollInterval)
	}
	for _, key := range keys {
		if member := tracking.members[key]; member != nil {
			revision := member.RevisionCounter
			member.LastRetrieved = &revision
		}
	}
	return len(keys) - len(outstanding), tracking.write()
}
