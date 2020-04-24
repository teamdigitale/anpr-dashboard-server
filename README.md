# ANPR Dashboard Server

The ANPR dashboard server is a [go-based application](https://golang.org/) that provides data related to the migration status of Italian municipalities to the National Registry.

The service exposes data, both via a REST API interface and through a GUI, that can be used to both download existing datasets and to upload new data in CSV format.

Some significant charts that consume these APIs and show the results can be seen on the [ANPR dashboard / stato migrazione website](https://stato-migrazione.anpr.it/).

## Main components

Some of the folders in this repository are particularly significant:

* **converter**: a tool to convert the data (schede) collected by [SOGEI](http://www.sogei.it/) into json files, by adding geolocation informations from Google maps.

* **site**: the website that shows the main statistical data, that allow users to download a csv file with the latest data and to upload a csv with new data

* **openapi**: the OpenAPI 3 specification of the APIs exposed

## GUI/dashboard

The application also exposes a web GUI that can be accessed from the root fo the website (/).

Through the GUI, it's possible to view the most common statistical data, download a CSV file with the latest datasets and upload new data.

## APIs

The service exposes both some public, as well some private APIs. Public APIs are described in the [OpenAPI 3 specification](openapi/anpr-dashboard.yaml).

For example, through APIs it's possible to retrieve the state of the migration to ANPR for single a municipality, or for all of them.

## Sandbox environments

For development purposes the application can also be run locally, directly on the developer machine, or in form of a Docker container. Following, both procedures are briefly explained.

### Run the project directly on the local machine

```shell
# Configuration
cd anpr-dashboard-server/server
mkdir creds
cp {THREE-CONFIG-FROM-ADMINS} creds
mkdir dashboard-subentro
cp {ANOTHER-CONFIG-FILE-FROM-ADMINS} subentroconfig.yaml dashboard-subentro/

# Build the project
make build

# Run the service
./server --mode=debug --config-file=/<PATH-TO-YOUR-WORKSPACE>/anpr-dashboard-server/dashboard-subentro/subentroconfig.yaml
```

### Run the project as a Docker container

A `Dockerfile` and a `docker-compose.yaml` files are in the root of the repository.

By default, the [docker-compose.yaml file](docker-compose.yaml) mounts some [exemplar configuration files](development_demo_data) -used by ANPR dashboard server- into the container. Check out what configurations get mounted and modify them as needed.

> NOTE: [cookie-creds/data.json](development_demo_data/vault/cookie_creds/data.json) is voluntarily left empty. New keys will be generated at every run.

Then, bring up the development environment in form of container, running:

```shell
docker-compose up [-d] [--build]
```

where:

* *-d* executes the container in background

* *--build* forces the container to re-build

The website and the APIs should now be accessible on port *8080*. While the GUI can be accessed at */*, APIs can be accessed under at */api*.
For example, you should be able to retrieve demo data used to build the dashboards at http://127.0.0.1:8080/api/dashboard/data.json

To bring down the test environment and remove the containers use

```shell
docker-compose down
```

## The cronjob Docker image and scripts

A custom script periodically runs to fetch the latest ANPR data and feed the dashboards. The script usually runs in form of a Docker container, on top of Kubernetes.

The script, the *Dockerfile* and the *docker-compose.yaml* files needed to build the container are located in the [cronjob](cronjob) folder of this repository.

## Query examples

The [query examples page](QUERY_EXAMPLES.md) provides some examples of relevant queries to extract useful informations from the database.

## How to contribute

Contributions are welcome! Feel free to open issues and submit a pull request at any time.

## License

Copyright (c) 2020 Presidenza del Consiglio dei Ministri

This program is free software: you can redistribute it and/or modify it under the terms of the GNU Affero General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version.

This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License along with this program.  If not, see <https://www.gnu.org/licenses/>.
