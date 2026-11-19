# sf-Compatible Mode Checklist

Reference: `sf` 2.138.6. Update this file as each item is finished.

## Step 1: Dispatch And Output
- [x] Binary name detection in `main.go`
- [x] sf root command, colon-form ids, legacy command aliases
- [x] Legacy flag names (`--targetusername`, `-u`, `--targetdevhubusername`, `--apiversion`)
- [x] `--json` envelope, error envelope, human error format, table renderer
- [x] `ErrorAndExit` handler hook
- [x] `version`

## Step 2: sf State
- [x] Config read/write (`~/.sf/config.json`, `<project>/.sf/config.json`, env)
- [x] Alias read/write (`~/.sfdx/alias.json`)
- [x] Auth file write and delete, field encryption
- [x] Org resolution (flag, env, local config, global config)
- [x] Save refreshed tokens to the sf auth file
- [x] `alias list`, `alias set`, `alias unset`
- [x] `config get`, `config list`, `config set`, `config unset`
- [x] `org list`, `org list auth`, `org display`, `org logout`

## Step 3: org
- [x] `org login web` (not exercised against a live org yet)
- [x] `org login jwt` (not exercised against a live org yet)
- [x] `org login access-token`
- [x] `org open` (format taken from the sf plugin source; the local test server has no single-access endpoint)
- [x] `org display user`
- [x] `org create scratch` (including definition file) (follows the sf plugin source; not compared with sf against a Dev Hub)
- [x] `org list limits` (format taken from the sf plugin source; the local test server has no limits resource)
- [x] `org list metadata`
- [x] `org list metadata-types`

## Step 4: data
- [x] `data query`
- [x] `data search` (format taken from the sf plugin source; the local test server has no search resource)
- [x] `data get record`, `data create record`, `data update record`, `data delete record`
- [x] `data import bulk`, `data update bulk`, `data upsert bulk`, `data delete bulk`, `data export bulk` (prints the last frame of the progress display once; sf redraws it while waiting)
- [x] `data import resume`, `data update resume`, `data upsert resume`, `data delete resume`, `data export resume`
- [x] `data bulk results`
- [ ] `data resume`
- [ ] `force data bulk delete`, `force data bulk upsert`, `force data bulk status`

## Step 5: apex, sobject, api, template
- [x] `apex run`
- [x] `apex run test` (`--output-dir` and `--detailed-coverage` are not supported)
- [x] `apex get log`, `apex list log`, `apex tail log` (formats taken from the sf plugin source; `apex tail log` is not exercised by a test)
- [x] `apex trace create`, `apex trace list`, `apex trace delete` (these exist in plugin-apex versions newer than the one bundled with sf 2.138.6)
- [x] `sobject describe`, `sobject list`
- [x] `api request rest`
- [x] `template generate apex class`, `template generate apex trigger`
- [x] `template generate visualforce page`, `template generate visualforce component`

## Step 6: project
- [x] Source collection for sf mode is built on `lib.PackageBuilder` in `sf/project_files.go`; the helpers in `command/` did not need to move
- [x] `project deploy start`, `project deploy validate` (queued output and progress layout compared with sf; the tables after a finished deploy follow the sf plugin source, because sf cannot finish a deploy against the local test server)
- [x] `project deploy quick`, `project deploy report`, `project deploy resume`, `project deploy cancel`
- [x] `project retrieve start` (failure output compared with sf; the result table follows the sf plugin source)
- [x] `project convert source`, `project convert mdapi`

## Step 7: package
- [x] `package create`, `package list` (follows the sf plugin source; not compared with sf against a Dev Hub)
- [x] `package install`, `package installed list`, `package uninstall` (follows the sf plugin source; not compared with sf against a Dev Hub)
- [x] `package version create`, `package version list`, `package version report`, `package version promote` (follows the sf plugin source; not compared with sf against a Dev Hub)

## Step 8: Documentation And Build
- [ ] `docs/sf-compatibility.md` (mapping both directions, missing commands)
- [ ] Generated `docs/sf/` from the sf command tree
- [ ] README section
- [ ] Makefile `sf` and `install-sf` targets
