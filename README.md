In this directory you can find everything that is needed for the ANPR dashboard.

Specifically:

1) **converter** - containing a tool to process the "schede" collected by SOGEI
   into json files, by adding geolocation information from google maps.

2) **site** - the site itself.



 ## Alcune query da eseguire sul server
Cerca fornitore per un COMUNE

 ```SELECT C.NAME,F.NAME as NOME_FORNITORE FROM COMUNE C
INNER JOIN FORNITORE F ON F.ID = C.ID_FORNITORE
WHERE UPPER(C.NAME)  LIKE  "%CARATE BRIANZA%"```

SELECT ID,NAME FROM FORNITORE WHERE  UPPER(FORNITORE.NAME) LIKE "%SYSTEM%"

```
Cambia fornitore per un comune
```
UPDATE COMUNE SET ID_FORNITORE = (SELECT ID FROM FORNITORE WHERE  UPPER(FORNITORE.NAME)="AP SYSTEMS") WHERE COMUNE.NAME= "CARATE BRIANZA"
```


Query estrazione del piano di subentro
```
SELECT COMUNE.NAME,
FORNITORE.NAME as FORNITORE,
COMUNE.POPOLAZIONE,  
date(SUBENTRO.RANGE_FROM,'unixepoch') as DATE_FROM,
date(SUBENTRO.RANGE_TO,'unixepoch') as DATE_TO

FROM COMUNE
INNER JOIN FORNITORE on FORNITORE.ID = COMUNE.ID_FORNITORE
INNER JOIN SUBENTRO on SUBENTRO.ID_COMUNE = COMUNE.ID ORDER BY SUBENTRO.RANGE_FROM ASC;
```

Per estrarre la query di subentro eseguire il comando dalla location del db di dashboard
```sqlite3 -header -csv db.sqlite <query.sql > subentro.csv```


Fornitori che hanno inserito alcuni comuni
```
SELECT FORNITORE.NAME as FORNITORE,
COUNT(FORNITORE.NAME)as NUMERO
FROM COMUNE
INNER JOIN FORNITORE on FORNITORE.ID = COMUNE.ID_FORNITORE
INNER JOIN SUBENTRO on SUBENTRO.ID_COMUNE = COMUNE.ID
WHERE COMUNE.DATA_SUBENTRO IS NULL
GROUP BY FORNITORE.NAME;
```

Somma dei comuni non subentrati
```
SELECT SUM(COMUNE.POPOLAZIONE)
FROM COMUNE
INNER JOIN FORNITORE on FORNITORE.ID = COMUNE.ID_FORNITORE
INNER JOIN SUBENTRO on SUBENTRO.ID_COMUNE = COMUNE.ID AND SUBENTRO.RANGE_TO <CAST(strftime('%s', '2017-12-31') AS INT) AND COMUNE.SUBENTRO_DATE IS NULL  
```

Inserimento di un COMUNE
```
INSERT INTO COMUNE (NAME,ID_FORNITORE,PROVINCIA,CODICE_ISTAT,POSTAZIONI,POPOLAZIONE,REGION,LAT,LON)
VALUES("BELVEDERE MARITTIMO",4,"CS","078015",4,9240,"CALABRIA",39.6332469,15.8417781);


INSERT INTO COMUNE (NAME,ID_FORNITORE,PROVINCIA,CODICE_ISTAT,POSTAZIONI,POPOLAZIONE,REGION,LAT,LON)
VALUES("MONTECCHIO EMILIA",3,"RE","035027",0,10622,"EMILIA ROMAGNA",44.7084791,10.4255221);

INSERT INTO COMUNE (NAME,ID_FORNITORE,PROVINCIA,CODICE_ISTAT,POSTAZIONI,POPOLAZIONE,REGION,LAT,LON)
VALUES("GARDA",23,"VR","023036",3,4126,"VENETO",45.5787644,10.6852263);



INSERT INTO COMUNE (NAME,ID_FORNITORE,PROVINCIA,CODICE_ISTAT,POSTAZIONI,POPOLAZIONE,REGION,LAT,LON)
VALUES("TRIBIANO",22,"MI","015222",2,3535,"LOMBARDIA",45.4126596,9.3673043);

ALTER TABLE COMUNE ADD  column POPOLAZIONE_AIRE INT

```

Update manuale delle date di subentro di un comune
```
UPDATE SUBENTRO SET  RANGE_FROM =  x
,RANGE_TO =  strftime('%s','2017-11-30 00:00:00')
,FINAL_DATE = strftime('%s','2017-11-27 00:00:00')
WHERE ID_COMUNE = (SELECT ID FROM COMUNE WHERE NAME="CASTELLEONE")
```

go ins```

**Metodo alternativo per l'installazione della propria sandbox**

*se volete mantenere il codice sorgente in un vostro workspace fuori dal $GOPATH...*


 1. **setup configurazioni varie e build del progetto**
 ```
 cd anpr-dashboard-server/server
 mkdir creds
 cp <TRE FILE DI CONFIGURAZIONE DA RICHIEDERE> creds
 mkdir dashboard-subentro
 cp <PATH ALTRO FILE DI CONFIG DA RICHIEDERE>subentroconfig.yaml dashboard-subentro/
 make build
 ```

 2. **run del server**
 ```
 ./server --mode=debug --config-file=/<PATH VOSTRO WORKSPACE>/anpr-dashboard-server/dashboard-subentro/subentroconfig.yaml
 ```

 3. **in caso ci fossero problemi con le dipendenze...**
 ```
git config --global --add url."git@github.com:".insteadOf "https://github.com/"
get ./...
 ```
e poi ripetere il punto 3

#ADD 3 luglio 2019
ALTER TABLE COMUNE ADD  column DATA_CESSAZIONE INT;
ALTER TABLE COMUNE ADD  column TIPO_CESSAZIONE TEXT;
ALTER TABLE COMUNE ADD  column COMUNE_CONFLUENZA TEXT;


#

