Core Principles:
Agent Role: Primarily inventory/data gathering, but also acts as a remote execution engine for provided PowerShell scripts (both standard inventory and custom checks).
External Correlation: The main correlation of inventory data (OS version, packages, patches) against vulnerability databases happens on the backend server.
Local Checks (Optional): Custom PowerShell scripts, executed by the agent, can perform specific local checks for vulnerabilities or misconfigurations if designed to do so.
Modularity (OS): The Go code orchestrating the scan will use OS-specific logic (runtime.GOOS) to determine how to gather data (e.g., which PowerShell scripts to run on Windows, different commands/methods on Linux/macOS).
Modularity (Scripts): All PowerShell scripts (inventory and custom) will be managed together, allowing engineers to easily add new custom checks.
Revised Windows Scan Architecture Plan:
PowerShell Script Repository:
Location: A designated directory within the project structure (e.g., scripts/powershell/windows/) containing all PowerShell scripts the agent might execute for Windows scans.
Contents:
Inventory Modules: Standardized scripts like Get-InventorySoftware.ps1, Get-InventoryPatches.ps1, Get-InventoryConfig.ps1, etc. These are designed to output structured JSON.
Custom Check Scripts: Any additional .ps1 files provided by engineers for specific checks (e.g., Check-CVE-2023-1234.ps1, Verify-AppLockerPolicy.ps1). Their output format might vary (simple text, boolean, or custom JSON).
Deployment: These scripts must be packaged and deployed alongside the agent binary so the agent can access and read their content.
internal:scan Command Parameters (args):
The command needs to be flexible. The args string passed to ScanCommand.Execute will need parsing. Examples:
internal:scan: Default behavior (e.g., run all standard inventory modules).
internal:scan --modules=software,patches: Run specific inventory modules.
internal:scan --scripts=Check-CVE-XYZ.ps1,Check-CustomPolicy.ps1: Run specific custom scripts from the repository.
internal:scan --all-modules --scripts=Check-CVE-XYZ.ps1: Run everything.
Parsing this argument string will happen within the Go ScanCommand.Execute method.
Go Orchestration (ScanCommand.Execute):
Parse Args: Determine which inventory modules and/or custom scripts are requested based on the args string.
Gather Basic Info: Collect OSInfo (OS, Version, IP, Hostname) using runtime and sysinfo helpers.
Initialize Result: Create an empty ScanResult struct (which will need fields for packages, patches, config checks, custom results, errors, etc.).
OS-Specific Logic (switch runtime.GOOS):
case "windows":
Iterate through the list of requested PowerShell scripts (both inventory modules and custom checks).
For each script name:
Construct the expected path to the script file within the deployed repository (e.g., ./scripts/powershell/windows/Get-InventorySoftware.ps1).
Read Script Content: Open and read the content of the .ps1 file into a string variable in Go.
Execute: Call shell.ExecuteScript(ctx, agentInfo.powerShellPath, scriptFileContent). This passes the content via stdin, which is safer and avoids issues with execution paths/permissions.
Process Result:
If it was a known inventory module (e.g., Get-InventorySoftware.ps1), attempt to unmarshal the stdout as JSON into the corresponding Go struct field within ScanResult (e.g., result.Packages). Handle JSON parsing errors.
If it was a custom script, store the raw execution outcome (stdout, stderr, exit code, execution error) in a map within ScanResult keyed by the script name (e.g., result.CustomResults["Check-CVE-XYZ.ps1"] = CustomScanOutput{...}).
Append any execution errors (err from ExecuteScript) or parsing errors to the ScanResult.ScanErrors list.
case "linux", "darwin": (To be detailed later)
Similar logic, but would likely call different helper functions (e.g., gatherLinuxPackages(), gatherMacOSPackages()) which might use exec.Command directly for native tools (dpkg, rpm, brew) instead of PowerShell.
Marshal Final Result: Convert the populated ScanResult Go struct (including OS info, inventory data, custom results, and any errors) into a final JSON output string.
Return: Return the JSON string and a nil error (unless the core orchestration fails). Errors encountered during individual script executions are reported within the JSON result.
Extended Result Structure (ScanResult):
Needs fields for each type of inventory data (packages, patches, config, network, users).
Needs a field to store results from custom scripts (e.g., CustomResults map[string]CustomScanOutput).
This architecture provides a clear separation: Go handles orchestration, cross-platform logic, and communication; PowerShell (on Windows) handles the OS-specific data gathering via a defined set of inventory scripts and provides an engine for running custom checks defined in separate script files. The agent delivers a structured JSON report containing all gathered information and any errors encountered during the scan.
