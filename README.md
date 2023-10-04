# autosys-nanny

## A command-line tool for managing services defined in yaml configuration file.

Now supports only Linux systems (RedHat, Ubuntu etc.).

### Build

`make`


### Flags:

| Long flag, short flag | Type | Default value | Required | Description |
| - | - | - | - | - |
| `--config`, `-c` | string | "" | **Yes** | Path to YAML file with services properties |
| `--force-restart`, `-r` | bool | false | No | Restart services even than they already running |
| `--list`, `-l` | bool | false | No | Only check services (without restart) and list them |
| `--log-file`, `-f` | string | "" | No | Path to log file |
| `--workers-num`, `-w` | int | 100 | No | Maximum number of concurrent workers for processing services |
| `--debug`, `-v` | bool | false | No | Enable debug mode |
| `--version` | bool | false | No | Show application version and exit |
| `--help` | bool | false | No | Show usage information and exit |


### Config file:

| Section | Parameter | Type | Default value | Required | Description |
| - | - | - | - | - | - |
| general | - | object | general | **Yes** | Main configuration common for all services (should be specified with port) |
| general | `mail_smtp_server` | string | "" | No | SMTP server for sending emails |
| general | `mail_auth_user` | string | "" | No | Mail user for authentication on SMTP server |
| general | `mail_auth_password` | string | "" | No | Mail password for authentication on SMTP server |
| general | `mail_address_from` | string | `${HOSTNAME}@${HOST_DOMAIN}` | No | Mail address in email's 'From:' field |
| general | `mail_subject_prefix` | string | `${HOSTNAME}` | No | Mail subject prefix |
| general | `mail_content_type` | string | "text/plain; charset=utf-8" | No | Mail content type (supported formats: "text/plain", "text/html") |
| general | `mailing_list` | []string | [] | No | List of emails to which script internal errors will be sent |
| services_list | - | []service | services_list | **Yes** | List of services to monitor and restart them |
| service | `process_name` | string | "" | **Yes** | Process name (with arguments) for search in process list |
| service | `description` | string | "" | No | Optional description of process |
| service | `disabled` | bool | false | No | Flag for disabling/enabling service |
| service | `start_cmd` | string | "" | **Yes** | Command to start service |
| service | `cmd_args` | []string | [] | No | Additional arguments for `start_cmd` command |
| service | stop_cmd | string | "" | No | Command to stop service |
| service | python_venv | string | "" | No | Path to python virtual environment |
| service | working_directory | string | "" | No | Path to working directory |
| service | pid_file | string | "" | No | Path to PID file |
| service | env_vars | []string | [] | No | Additional environment variables |
| service | mailing_list | []string | [] | No |  List of emails to which service errors will be sent |


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
