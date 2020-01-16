# ANPR Dashboard Server

The ANPR dashboard server is a go-based application that provides data related to the migration status of Italian municipalities to the National Registry.

The service exposes data both through an API interface and through a UI, through which it's possible to both download existing datasets and upload new data in CSV format.

## Main components

Some of the folders in this repository are particularly significant:

* **converter**: a tool to convert the data (schede) collected by [SOGEI](http://www.sogei.it/) into json files, by adding geolocation informations from Google maps.

* **site**: the website that shows the main statistical data, that allow users to download a csv file with the latest data and to upload a csv with new data

* **openapi**: the OpenAPI 3 specification of the APIs exposed

## GUI/dashboard

The application also exposes a web-based GUI that can be accessed from website root ('/') once the server is running.

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

Search the supplier name (*fornitore*), given a municipality name (*comune*).

```sql
SELECT C.NAME, F.NAME as NOME_FORNITORE FROM COMUNE C
INNER JOIN FORNITORE F ON F.ID=C.ID_FORNITORE
WHERE UPPER(C.NAME) LIKE  "%CARATE BRIANZA%"

SELECT ID, NAME FROM FORNITORE WHERE UPPER(FORNITORE.NAME) LIKE "%SYSTEM%"
```

Change the supplier name (*fornitore*) for a given municipality (*comune*).

```sql
UPDATE COMUNE SET ID_FORNITORE = (SELECT ID FROM FORNITORE WHERE  UPPER(FORNITORE.NAME)="AP SYSTEMS") WHERE COMUNE.NAME="CARATE BRIANZA"
```

Extract the municipality takeover plan

```sql
SELECT COMUNE.NAME,
FORNITORE.NAME as FORNITORE,
COMUNE.POPOLAZIONE,  
date(SUBENTRO.RANGE_FROM, 'unixepoch') as DATE_FROM,
date(SUBENTRO.RANGE_TO, 'unixepoch') as DATE_TO
FROM COMUNE
INNER JOIN FORNITORE on FORNITORE.ID=COMUNE.ID_FORNITORE
INNER JOIN SUBENTRO on SUBENTRO.ID_COMUNE=COMUNE.ID ORDER BY SUBENTRO.RANGE_FROM ASC;
```

To extract the municipality takeover query

```shell
sqlite3 -header -csv db.sqlite < query.sql > subentro.csv
```

Get the list of suppliers that still haven't helped any municipality

```sql
SELECT FORNITORE.NAME as FORNITORE,
COUNT(FORNITORE.NAME) as NUMERO
FROM COMUNE
INNER JOIN FORNITORE on FORNITORE.ID=COMUNE.ID_FORNITORE
INNER JOIN SUBENTRO on SUBENTRO.ID_COMUNE=COMUNE.ID
WHERE COMUNE.DATA_SUBENTRO IS NULL
GROUP BY FORNITORE.NAME;
```

Sum of the municipalities that have not yet taken over

```sql
SELECT SUM(COMUNE.POPOLAZIONE)
FROM COMUNE
INNER JOIN FORNITORE on FORNITORE.ID=COMUNE.ID_FORNITORE
INNER JOIN SUBENTRO on SUBENTRO.ID_COMUNE=COMUNE.ID AND SUBENTRO.RANGE_TO <CAST(strftime('%s', '2017-12-31') AS INT) AND COMUNE.SUBENTRO_DATE IS NULL  
```

Some examples of municipalities insertions:

```sql
INSERT INTO COMUNE (NAME, ID_FORNITORE, PROVINCIA, CODICE_ISTAT, POSTAZIONI, POPOLAZIONE, REGION, LAT, LON)
VALUES("BELVEDERE MARITTIMO", 4, "CS", "078015", 4, 9240, "CALABRIA", 39.6332469, 15.8417781);

INSERT INTO COMUNE (NAME, ID_FORNITORE, PROVINCIA, CODICE_ISTAT, POSTAZIONI, POPOLAZIONE, REGION, LAT, LON)
VALUES("MONTECCHIO EMILIA", 3, "RE", "035027", 0, 10622, "EMILIA ROMAGNA", 44.7084791, 10.4255221);

INSERT INTO COMUNE (NAME, ID_FORNITORE, PROVINCIA, CODICE_ISTAT, POSTAZIONI, POPOLAZIONE, REGION, LAT, LON)
VALUES("GARDA", 23, "VR", "023036", 3, 4126, "VENETO", 45.5787644, 10.6852263);

INSERT INTO COMUNE (NAME, ID_FORNITORE, PROVINCIA, CODICE_ISTAT, POSTAZIONI, POPOLAZIONE, REGION, LAT, LON)
VALUES("TRIBIANO", 22, "MI", "015222", 2, 3535, "LOMBARDIA", 45.4126596, 9.3673043);
```

Update the takeover dates for a specific municipality

```sql
UPDATE SUBENTRO
SET RANGE_FROM=x, RANGE_TO=strftime('%s','2017-11-30 00:00:00'), FINAL_DATE=strftime('%s','2017-11-27 00:00:00')
WHERE ID_COMUNE=(SELECT ID FROM COMUNE WHERE NAME="CASTELLEONE")
```
