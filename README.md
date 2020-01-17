# ANPR Dashboard Server

The ANPR dashboard server is a go-based application that provides data related to the migration status of Italian municipalities to the National Registry.

The service exposes data both via API interface and through a UI, that can be used to both download existing datasets and to upload new data in CSV format.

## Main components

Some of the folders in this repository are particularly significant:

* **converter**: a tool to convert the data (schede) collected by [SOGEI](http://www.sogei.it/) into json files, by adding geolocation informations from Google maps.

* **site**: the website that shows the main statistical data, that allow users to download a csv file with the latest data and to upload a csv with new data

* **openapi**: the OpenAPI 3 specification of the APIs exposed

## GUI/dashboard

The application also exposes a web UI that can be accessed from website root ('/') once the server is running.

Through the GUI it's possible to view the main statistical data, download a csv file with the latest datasets and to upload a csv with new data.

## APIs

The service exposes both some public, as well some private APIs. The public ones are described in the [OpenAPI 3 specification](openapi/anpr-dashboard.yaml).

Through the APIs, it's -for example- possible to retrieve the state of the migration to ANPR for single a municipality or for all of them together.

## Sandbox environments

For development purposes the project can also be run locally, directly on the developer machine, or in form of a Docker container. Following, both procedures are explained.

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

A `Dockerfile` and a `docker-compose.yaml` files are in the root of this repository.

Create a new *vault* directory in the root of this repository. Put your development configurations and credentials in it. Make sure they reference */srv/db/sqlite.db*. Put the sqlite database in *vault/db* and name it *sqlite.db*

To build the local test environment run:

```shell
UID=$UID docker-compose up -d
```

The website and the APIs should now be accessible on port *8080*. While the website is accessible at the root, the APIs can be accessed under the path */api*

To bring down the test environment and remove the containers use

```shell
docker-compose down
```

## Query examples

The [query examples page](QUERY_EXAMPLES.md) provides examples around relevant queries to extract useful informations from the database.

## How to contribute

Contributions are welcome! Feel free to open issues and submit a pull request at any time.

## License

Copyright (c) 2019 Presidenza del Consiglio dei Ministri

This program is free software: you can redistribute it and/or modify it under the terms of the GNU Affero General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version.

This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License along with this program.  If not, see <https://www.gnu.org/licenses/>.
