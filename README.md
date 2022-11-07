# grafana-sidecar-users-yaml
Sidecar for Grafana that reads users from YAML file and updates it in the database

### Building
To build the binary just run `go build`

#### Building Docker image
To build the docker image use `Dockerfile`:

    docker build -t $YOUR_TAG .

### Running
Check the below help message for `start` command:

    Usage:  grafana-sidecar-users-yaml start [FLAGS]
    
    Starts the daemon
    
    Required flags: 
      -c,	 --config config 	YAML file with users
    
    Optional flags: 
      -i,	 --ignore_errors  	Ignore errors and continue


#### Ignoring errors
The program runs an infinite loop in which it loads the configuration file and based on it, it performs `UPDATE` queries. The configuration file is loaded with each loop iteration and in case of any error (syntax error, db connection etc.) it will crash. You can set it to ignore the errors and continue working.

### Configuration file
Program requires a configuration file, which is passed with `-c`. An example of one can be found in the `config.yaml.example` file, or below:

    version: "1"
    database: /path/to/grafana.db
    dry_run: no
    run_once: no
    sleep: 10
    orgs:
      - id: 1
        admins:
          - login: mikogs
        editors:
          - login: editor1
          - login: editor2
        viewers:
          - login: viewer1
          - login: viewer2

| Key | Description | Value |
|--------------|-----------|------------|
| database | Path to grafana.db SQLite database | string |
| dry_run | When true, no database queries will be executed | boolean |
| run_once | When true, the program will run once and exit. Otherwise it will start as a daemon | boolean |
| sleep | Seconds to wait before loop iterations | integer |
| orgs | List of organisations with its admins, editors and viewers | [org] |

`org` has the following fields:
| Key | Description | Value |
|---- | ----------- | ----- |
| id | Organisation ID | integer |
| admins | List of users | `[{ "login": string}]` |
| editors | List of users |`[{ "login": string}]` |
| viewers | List of users | `[{ "login": string}]` |

### Example usage in Kubernetes
Do the following stes to add the program as a sidecar to your Grafana running in Kubernetes cluster:

* add configuration as ConfigMap
* add a sidecar (with built docker image) to your Grafana pod, with ConfigMap mounted in a path that is accessible by the program, and set `-c` to the configuration file in that path
* ensure your Kubernetes cluster automatically updates volumes mounted from ConfigMaps

In the below example, you can see the sidecar added to Grafana helm chart:

    extraConfigmapMounts:
      - name: sc-users
        configMap: grafana-sc-users
        mountPath: /etc/grafana/provisioning/sc-users
        subPath: ''
  
    extraContainers: |-
      - name: grafana-sc-users
        image: mikogs/grafana-sidecar-users-yaml:0.1.0
        imagePullPolicy: Always
        args:
          - "start"
          - "-i"
          - "-c"
          - "/etc/grafana/provisioning/sc-users/users.yaml"
        volumeMounts:
          - mountPath: /etc/grafana/provisioning/sc-users
            name: sc-users
          - mountPath: /var/lib/grafana
            name: storage

