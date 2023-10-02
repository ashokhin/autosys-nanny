# autosys-nanny

## A command-line tool for managing services defined in yaml configuration file.

Now supports only Linux systems (RedHat, Ubuntu etc.).

### Flags:
| Long flag, short flag | Type | Default value | Required | Description |
| - | - | - | - | - |
| `--config`, `-c` | string | "" | Yes | Path to YAML file with services properties |
| `--force-restart`, `-r` | bool | false | No | Restart services even than they already running |
| `--list`, `-l` | bool | false | No | Only check services (without restart) and list them |
| `--log-file`, `-f` | String | "" | No | Path to log file |
| `--workers-num`, `-w` | int | 100 | No | Maximum number of concurrent workers for processing services |
| `--debug`, `-v` | bool | false | No | Enable debug mode |
| `--version` | bool | false | No | Show application version and exit |
| `--help` | bool | false | No | Show usage information and exit |


### Config file:
| Section | Parameter | Type | Default value | Required | Description |
| - | - | - | - | - | - |
| general | - | object | general | Yes | Main configuration common for all services (should be specified with port) |
| general | mail_smtp_server | string | "" | No | SMTP server for sending emails |
| general | mail_auth_user | string | "" | No | Mail user for authentication on SMTP server |
| general | mail_auth_password | string | "" | No | Mail password for authentication on SMTP server |
| general | mail_address_from | string | `${HOSTNAME}@${HOST_DOMAIN}` | No | Mail address in email's 'From:' field |
| general | mail_content_type | string | "text/plain; charset=utf-8" | No | Mail content type (supported formats: "text/plain", "text/html")

### Before use:
Create configuration file with services.
Example [services.yaml.example](./services.yaml.example)

### Usage examples:

#### List services and exit, output to stdout:


### TODO
- finish configuration file description
- add commands examples and output examples
