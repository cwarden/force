# sf-Compatible Mode Checklist

Reference: `sf` 2.138.6. Update this file as each item is finished.

## Step 1: Dispatch And Output
- [x] Binary name detection in `main.go`, and the `force --sf` flag
- [x] sf root command, colon-form ids, legacy command aliases
- [x] Legacy flag names (`--targetusername`, `-u`, `--targetdevhubusername`, `--apiversion`)
- [x] Deprecation warning when a legacy command id is used (`force:org:list`)
- [x] Legacy flag names of each command (`--classname`, `--outputdir`, ...; 488 names in `sf/flagaliases.go`) and their deprecation warnings
- [x] Suggestion of the closest username or alias when an org name has no authorization
- [x] Error when a flag that takes one value is given twice (`Flag --name can only be specified once`)
- [x] `--json` envelope, error envelope, human error format, table renderer
- [x] `ErrorAndExit` handler hook
- [x] `version`, `commands`

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
- [x] `data resume` (follows the sf plugin source)
- [x] `force data bulk delete`, `force data bulk upsert`, `force data bulk status` (follow the sf plugin source)

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
- [x] `docs/sf-compatibility.md` (mapping both directions, missing commands); `TestCompatibilityDocumentListsEveryCommand` fails when a command has no row
- [x] Generated `docs/sf/` from the sf command tree (`make docs`)
- [x] README section
- [x] Makefile `install-sf` target. There is no `sf` build target: `go build -o sf` would write into the `sf/` package directory

## Unimplemented Flags

Flags of implemented commands that return an error. A flag marked "not defined" is reported as a nonexistent flag; the others are reported as not supported by this mode.

- [ ] `apex run`: `--debug-level`, `--category-level` (not defined)
- [x] `apex run test`: `--output-dir`, `--detailed-coverage`
- [ ] `apex tail log`: `--debug-level`
- [x] `api request rest`: `--file` (the `formdata` body mode is not supported)
- [ ] `commands`: `--columns`, `--deprecated`, `--extended`, `--hidden`, `--no-truncate`, `--sort`, `--tree` (not defined)
- [ ] `data delete resume`, `data upsert resume`: `--target-org` (not defined; the org comes from the job cache)
- [ ] `org create scratch`: `--async`, `--client-id`, `--no-ancestors`, `--track-source`, `--source-org`, `--snapshot`
- [ ] `org login web`: `--browser`, `--client-app`, `--username`, `--scopes`
- [ ] `org open`: `--private`, `--browser`, `--source-file`
- [ ] `package create`: `--no-namespace`, `--org-dependent`, `--error-notification-username`
- [ ] `package install`: `--publish-wait`, `--skip-handlers`
- [ ] `package version create`: `--installation-key`, `--definition-file`, `--branch`, `--post-install-script`, `--post-install-url`, `--releasenotes-url`, `--uninstall-script`, `--language`, `--generate-pkg-zip`, `--skip-ancestor-check`, `--verbose`
- [ ] `package update`: `--skip-ancestor-check`
- [ ] `package version create list`: `--show-conversions-only`, `--verbose`
- [ ] `package version list`: `--show-conversions-only`
- [x] `project convert mdapi`: `--manifest`, `--metadata-dir`, `--metadata`
- [x] `project convert source`: `--package-name`
- [ ] `project deploy start`, `project deploy validate`: `--coverage-formatters`, `--junit`, `--results-dir` (`--pre-destructive-changes` and `--post-destructive-changes` are done)
- [ ] `project deploy report`, `project deploy resume`: `--coverage-formatters`, `--junit`, `--results-dir` (not defined)
- [x] `project deploy quick`, `report`, `resume`, `cancel`: `--use-most-recent`
- [ ] `project generate manifest`: `--from-org`, `--include-packages`
- [ ] `project retrieve start`: `--single-package`; more than one `--package-name`

## Flags Accepted Without Effect

- [ ] `alias unset`: `--no-prompt` (there is no prompt)
- [ ] `apex tail log`: `--color`
- [ ] `org create scratch`: `--wait` (the wait is the one in lib: up to 10 minutes)
- [x] `org display`: `--verbose` (no sfdxAuthUrl is shown) — now adds the redacted Sfdx Auth Url row, as sf does
- [ ] `org list`: `--verbose`, `--clean`, `--no-prompt`
- [ ] `org login access-token`: `--no-prompt`
- [ ] `org logout`: `--client-app`
- [ ] `package install`: `--no-prompt` (remote site settings are enabled without asking)
- [ ] `package version create`: `--wait` (the command always waits), `--installation-key-bypass`
- [ ] `package version report`: `--verbose`
- [ ] `project deploy start`, `project deploy validate`: `--verbose`, `--ignore-conflicts`, `--single-package`
- [ ] `project deploy quick`, `project deploy resume`: `--verbose`
- [ ] `project retrieve start`: `--ignore-conflicts`, `--wait`
- [ ] `template generate project`: the `agent`, `nativemobile`, `reactinternalapp` and `reactexternalapp` templates
- [x] `template generate apex trigger`, `template generate visualforce page`, `template generate visualforce component`: `--template` (each command has one template; another value is an error, as in sf)

## Unimplemented Behavior Of Implemented Commands

- [x] `template generate apex class`: templates other than `DefaultApexClass`
- [ ] `org create scratch`: definition file key `objectSettings`, and keys other than `orgName`, `edition`, `country`, `language`, `adminEmail`, `description`, `hasSampleData`, `username`, `release`, `namespace`, `duration`, `durationDays`, `features`, `settings`
- [ ] `org display`: scratch org fields other than the Dev Hub username and expiration date
- [x] `data query`: sf's layout for child records of a subquery, and columns from the column metadata of the query (`columns=true`)
- [ ] `project convert source`: the order of elements in a composed custom object (sf writes the fields and other child components after the elements of the object file; this mode writes all elements in alphabetical order)
- [ ] `project convert mdapi --metadata`, `--manifest`: a child component named on its own, such as `CustomField:Invoice__c.Amount__c` (only whole objects are selected)
- [ ] Unknown metadata type name: the "Did you mean one of the following types?" line (needs sf's list of all type names)
- [ ] `package version list`: ancestor version numbers and code coverage
- [ ] `force data bulk delete`, `force data bulk upsert`: batches of 10,000 records (the file is sent as one batch)
- [ ] `project retrieve start`: `fileProperties` in the JSON result
- [ ] `api request rest --include`: headers in the order received
- [ ] `config set`, `config get`: the prompt that suggests a similar config name
- [ ] A topic given without a command (`sf org`): sf's prompt to pick a command
- [ ] Error envelopes: `stack`, `cause`, and `data` fields other than `errorCode` and `message`
- [ ] Redrawn progress display for bulk, deploy, retrieve and scratch org commands

## Unimplemented Features Shared By Several Commands

- [x] `--flags-dir` on every command
- [x] `--use-most-recent` on the deploy commands (needs `~/.sf/deploy-cache.json`)
- [ ] Source tracking
  - [x] Read and write `maxRevision.json` and the `localSourceTracking` git repository in the formats of sf (sf read the files this mode wrote)
  - [x] `org enable tracking`, `org disable tracking`, `project reset tracking`, `project delete tracking`
  - [ ] `project deploy preview`, `project retrieve preview`
  - [ ] Deploy and retrieve without naming components, and update the tracking files after a deploy or retrieve
  - [ ] Conflict detection (`--ignore-conflicts`)
  - [ ] Detection of moved files
  - [ ] Write `maxRevision.json` as fileVersion 1 when `SF_SOURCE_TRACKING_FILE_VERSION=1` (the file is read in both versions)
- [x] `.forceignore` (sf reports an error when a content file is ignored and its `-meta.xml` file is not; this mode leaves both out)
- [x] Destructive changes on deploy
- [ ] Coverage and JUnit result files on the deploy commands
- [x] Shell completion for the sf command tree (`sf autocomplete`)

## Needs A Decision

- [x] Template files: the 192 files of the salesforcedx-templates repository (tag 66.9.1) are embedded as compressed archives in `sf/templates`, with the license in `third-party-licenses.txt` and in every release zip. The React app templates of the npm package are not in that repository and are not included

## Not Yet Verified

- [ ] Run `org login web` against an org
- [ ] Run `org login jwt` against an org
- [ ] Compare the commands marked "follows the sf plugin source" with sf output against an org or Dev Hub

## Unimplemented Commands

163 commands of sf 2.138.6 and its bundled and install-on-first-use plugins. `docs/sf-compatibility.md` says what each needs. Commands under "dev" and most under "CLI management" manage the Node.js CLI and its plugins and do not apply to force.

### org (16 open, 11 done)

- [x] `org assign permset` (follows the sf plugin source)
- [x] `org assign permsetlicense` (follows the sf plugin source)
- [x] `org auth show-access-token`
- [x] `org auth show-sfdx-auth-url`
- [x] `org auth show-user-password` (follows the sf plugin source)
- [ ] `org create agent-user`
- [ ] `org create sandbox`
- [ ] `org create shape`
- [ ] `org create snapshot`
- [ ] `org create user`
- [ ] `org delete sandbox`
- [x] `org delete scratch` (follows the sf plugin source)
- [ ] `org delete shape`
- [ ] `org delete snapshot`
- [x] `org disable tracking`
- [x] `org enable tracking`
- [x] `org generate password` (follows the sf plugin source)
- [ ] `org get snapshot`
- [ ] `org list shape`
- [ ] `org list snapshot`
- [x] `org list sobject record-counts` (follows the sf plugin source)
- [ ] `org list users`
- [x] `org login sfdx-url` (follows the sf plugin source)
- [ ] `org open agent`
- [ ] `org refresh sandbox`
- [ ] `org resume sandbox`
- [ ] `org resume scratch`

### project (9 open, 4 done)

- [ ] `project convert source-behavior`
- [ ] `project delete source`
- [x] `project delete tracking`
- [ ] `project deploy pipeline quick`
- [ ] `project deploy pipeline report`
- [ ] `project deploy pipeline resume`
- [ ] `project deploy pipeline start`
- [ ] `project deploy pipeline validate`
- [ ] `project deploy preview`
- [x] `project generate manifest` (`--from-org` and `--include-packages` are not supported)
- [x] `project list ignored`
- [x] `project reset tracking`
- [ ] `project retrieve preview`

### data (0 open, 3 done)

- [x] `data create file` (follows the sf plugin source)
- [x] `data export tree`
- [x] `data import tree` (follows the sf plugin source)

### apex (1 open, 1 done)

- [ ] `apex get compile-status`
- [x] `apex get test`

### logic (2 open, 0 done)

- [ ] `logic get test`
- [ ] `logic run test`

### flow (2 open, 0 done)

- [ ] `flow get test`
- [ ] `flow run test`

### api (0 open, 1 done)

- [x] `api request graphql` (follows the sf plugin source)

### schema (3 open, 1 done)

- [ ] `schema generate field`
- [ ] `schema generate platformevent`
- [ ] `schema generate sobject`
- [x] `schema generate tab`

### template (3 open, 8 done)

- [x] `template generate analytics template`
- [ ] `template generate digital-experience site`
- [ ] `template generate flexipage`
- [x] `template generate lightning app`
- [x] `template generate lightning component`
- [x] `template generate lightning event`
- [x] `template generate lightning interface`
- [x] `template generate lightning test`
- [x] `template generate project` (the `agent`, `nativemobile` and React app templates are not supported)
- [x] `template generate static-resource`
- [ ] `template generate ui-bundle`

### package (8 open, 8 done)

- [ ] `package convert`
- [x] `package delete` (follows the sf plugin source)
- [x] `package install report` (follows the sf plugin source)
- [ ] `package push-upgrade abort`
- [ ] `package push-upgrade list`
- [ ] `package push-upgrade report`
- [ ] `package push-upgrade schedule`
- [x] `package uninstall report` (follows the sf plugin source)
- [x] `package update` (follows the sf plugin source)
- [x] `package version create list` (follows the sf plugin source)
- [x] `package version create report` (follows the sf plugin source)
- [x] `package version delete` (follows the sf plugin source)
- [ ] `package version displayancestry`
- [ ] `package version displaydependencies`
- [ ] `package version retrieve`
- [x] `package version update` (follows the sf plugin source)

### package1 (4 open, 0 done)

- [ ] `package1 version create`
- [ ] `package1 version create get`
- [ ] `package1 version display`
- [ ] `package1 version list`

### cmdt (1 open, 4 done)

- [x] `cmdt generate field`
- [ ] `cmdt generate fromorg`
- [x] `cmdt generate object`
- [x] `cmdt generate record`
- [x] `cmdt generate records`

### community (3 open, 0 done)

- [ ] `community create`
- [ ] `community list template`
- [ ] `community publish`

### lightning (3 open, 0 done)

- [ ] `lightning dev app`
- [ ] `lightning dev component`
- [ ] `lightning dev site`

### agent (33 open, 0 done)

- [ ] `agent activate`
- [ ] `agent adl create`
- [ ] `agent adl delete`
- [ ] `agent adl file add`
- [ ] `agent adl file delete`
- [ ] `agent adl file list`
- [ ] `agent adl get`
- [ ] `agent adl list`
- [ ] `agent adl status`
- [ ] `agent adl update`
- [ ] `agent adl upload`
- [ ] `agent create`
- [ ] `agent deactivate`
- [ ] `agent generate agent-spec`
- [ ] `agent generate authoring-bundle`
- [ ] `agent generate template`
- [ ] `agent generate test-spec`
- [ ] `agent preview`
- [ ] `agent preview end`
- [ ] `agent preview send`
- [ ] `agent preview sessions`
- [ ] `agent preview start`
- [ ] `agent publish authoring-bundle`
- [ ] `agent test create`
- [ ] `agent test list`
- [ ] `agent test results`
- [ ] `agent test resume`
- [ ] `agent test run`
- [ ] `agent test run-eval`
- [ ] `agent trace delete`
- [ ] `agent trace list`
- [ ] `agent trace read`
- [ ] `agent validate authoring-bundle`

### force (4 open, 0 done)

- [ ] `force lightning lwc test create`
- [ ] `force lightning lwc test run`
- [ ] `force lightning lwc test setup`
- [ ] `force package push-upgrade list`

### ui-bundle (1 open, 0 done)

- [ ] `ui-bundle dev`

### dev (6 open, 0 done)

- [ ] `dev audit messages`
- [ ] `dev convert messages`
- [ ] `dev convert script`
- [ ] `dev generate command`
- [ ] `dev generate flag`
- [ ] `dev generate plugin`

### CLI management (22 open, 1 done)

- [x] `autocomplete` (instructions for cobra completion scripts; the mechanism differs from sf)
- [ ] `doctor`
- [ ] `help`
- [ ] `info releasenotes display`
- [ ] `plugins`
- [ ] `plugins add`
- [ ] `plugins discover`
- [ ] `plugins inspect`
- [ ] `plugins install`
- [ ] `plugins link`
- [ ] `plugins remove`
- [ ] `plugins reset`
- [ ] `plugins trust allowlist add`
- [ ] `plugins trust allowlist list`
- [ ] `plugins trust allowlist remove`
- [ ] `plugins trust verify`
- [ ] `plugins uninstall`
- [ ] `plugins unlink`
- [ ] `plugins update`
- [ ] `search`
- [ ] `update`
- [ ] `whatsnew`
- [ ] `which`
