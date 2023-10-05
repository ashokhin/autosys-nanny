# autosys-nanny

## A command-line tool for managing services defined in yaml configuration file.

Now supports only Linux systems (RedHat, Ubuntu etc.).

### Build

`make`


### Flags:

| Long flag, short flag<br>_Type_ | Required<br>_Default value_ | Description |
|---|---|---|
| `--config`, `-c`<br>_string_ | **Yes**<br>_""_ | Path to YAML file with services properties |
| `--force-restart`, `-r`<br>_bool_ | No<br>_false_ | Restart services even than they already running |
| `--list`, `-l`<br>_bool_ | No<br>_false_ | Only check services (without restart) and list them |
| `--log-file`, `-f`<br>_string_ | No<br>_""_ | Path to log file |
| `--workers-num`, `-w`<br>_int_ | No<br>_100_ | Maximum number of concurrent workers for processing services |
| `--debug`, `-v`<br>_bool_ | No<br>_false_ | Enable debug mode |
| `--version`<br>_bool_ | No<br>_false_ | Show application version and exit |
| `--help`<br>_bool_ | No<br>_false_ | Show usage information and exit |


### Config file:

| Section | Parameter<br>_Type_ | Required<br>_Default value_ | Description |
|---|---|---|---|
| general | `-`<br>_object_ | **Yes**<br>_-_ | Main configuration common for all services (should be specified with port) |
| general | `mail_smtp_server`<br>_string_ | No<br>_""_ | SMTP server for sending emails |
| general | `mail_auth_user`<br>_string_ | No<br>_""_ | Mail user for authentication on SMTP server |
| general | `mail_auth_password`<br>_string_ | No<br>_""_ | Mail password for authentication on SMTP server |
| general | `mail_address_from`<br>_string_ | No<br>_`${HOSTNAME}@${HOST_DOMAIN}`_ | Mail address in email's 'From:' field |
| general | `mail_subject_prefix`<br>_string_ | No<br>_`${HOSTNAME}`_ | Mail subject prefix |
| general | `mail_content_type`<br>_string_ | No<br>_"text/plain; charset=utf-8"_ | Mail content type (supported formats: "text/plain", "text/html") |
| general | `mailing_list`<br>_[]string_ | No<br>_[]_ | List of emails to which script internal errors will be sent |
| services_list | `-`<br>_[]service_ | **Yes**<br>_services_list_ | List of services to monitor and restart them |
| service | `process_name`<br>_string_ | **Yes**<br>_""_ | Process name (with arguments) for search in process list |
| service | `description`<br>_string_ | No<br>_""_ | Optional description of process |
| service | `disabled`<br>_bool_ | No<br>_false_ | Flag for disabling/enabling service |
| service | `start_cmd`<br>_string_ | **Yes**<br>_""_ | Command to start service |
| service | `cmd_args`<br>_[]string_ | No<br>_[]_ | Additional arguments for `start_cmd` command |
| service | `stop_cmd`<br>_string_ | No<br>_""_ | Command to stop service |
| service | `python_venv`<br>_string_ | No<br>_""_ | Path to python virtual environment |
| service | `working_directory`<br>_string_ | No<br>_""_ | Path to working directory |
| service | `pid_file`<br>_string_ | No<br>_""_ | Path to PID file |
| service | `env_vars`<br>_[]string_ | No<br>_[]_ | Additional environment variables |
| service | `mailing_list`<br>_[]string_ | No<br>_[]_ | List of emails to which service errors will be sent |


> [!WARNING] 
> Before use create configuration file with services.
> Example [services.yaml.example](./services.yaml.example)


#### Minimal configuration file example:

```
general:
services_list:
  - process_name: "service2.py"
    start_cmd: "/usr/bin/python3 /opt/nanny/service2.py"
```


#### Usage examples:

##### Check services and start if enabled but stopped, output to stdout:

`./autosys_nanny --config=./services.yaml`


##### Check services and restart even they are running, output with debug information to .log file:

`./autosys_nanny --config=./services.yaml --force-restart --debug --log-file=./nanny.log`


##### List services and exit, output to stdout:

`./autosys_nanny --config=./services.yaml --list`


### TODO

- [ ] add output examples
