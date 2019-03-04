In this directory you can find everything that is needed for the ANPR dashboard.

Specifically:

1) **converter** - containing a tool to process the "schede" collected by SOGEI
   into json files, by adding geolocation information from google maps.

2) **site** - the site itself.


**Come aggiornare il file di configurazione della dashboard**

E' necessario scaricare il repository di produzione che é

https://github.com/teamdigitale/production

Prima di dare qualsiasi comando con ANSIBLE
**é necessario aggiornare il repository locale**

Il file di configurazione é
```
ansible/roles/teamdigitale_dashboard_anpr/templates/config.yaml.j2
```
Il comando ansible va lanciato dalla stessa directory "ansible".

Il comando da lanciare é il seguente e la password é recuperabile dal documento (https://docs.google.com/document/d/1rqy4uVXm0OeYuJXjjbUNnXNsOz483he7CDYfA7M6g6s/edit) ad accesso limitato

```
ansible-playbook site.yml --limit dashboard_anpr --diff --ask-vault-pass --tags "anpr"
```

Dry-run:
```
ansible-playbook site.yml --limit dashboard_anpr --diff --ask-vault-pass --check --tags "anpr"
```

Nel lanciare la dashboard collegati come root sul server in modalità ssh
```
ssh root@dashboard.anpr.it
```
Per rilanciare e dashboard (dashboard.anpr.it)

 ```stop dashboard && start dashboard```


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

1. **cloning del repo**
 ```
 cd <PATH VOSTRO WORKSPACE>
 ssh-agent bash -c 'ssh-add ~/.ssh/id_rsa; git clone git@github.com:teamdigitale/anpr-dashboard-server.git'
 ```
*aggiustate il path della vostra chiave ssh...*

 2. **setup configurazioni varie e build del progetto**
 ```
 cd anpr-dashboard-server/server
 mkdir creds
 cp <TRE FILE DI CONFIGURAZIONE DA RICHIEDERE> creds
 mkdir dashboard-subentro
 cp <PATH ALTRO FILE DI CONFIG DA RICHIEDERE>subentroconfig.yaml dashboard-subentro/
 make build
 ```

 3. **run del server**
 ```
 ./server --mode=debug --config-file=/<PATH VOSTRO WORKSPACE>/anpr-dashboard-server/dashboard-subentro/subentroconfig.yaml
 ```

 4. **in caso ci fossero problemi con le dipendenze...**
 ```
git config --global --add url."git@github.com:".insteadOf "https://github.com/"
get ./...
 ```
e poi ripetere il punto 3



#

