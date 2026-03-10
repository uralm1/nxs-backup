## 2. How to configure ##

After nxs-backup installation to a server, you need to generate configuration as the config does not
appear automatically.
nxs-backup configuration files are located in the /etc/nxs-backup/ directory by default.
The basic configuration has only the main configuration file nxs-backup.yml and an empty
subdirectory conf.d, where files with job descriptions should be stored (one file per job).
If the main config file does not exist, you will be prompted to add it at the first startup.
All configuration files are in YAML format.

This is an example of an empty main config:

/etc/nxs-backup/nxs-backup.yml
```yaml
server_name: localhost
project_name: My Best Project

loglevel: info

notifications:
  mail:
    enabled: false
  webhooks: []
storage_connects: []
jobs: []
include_job_configs: [ "conf.d/*.yml" ]
```

You can generate a configuration file by running nxs-backup with the generate command and the options:

1. Backup type `-T [--backup-type] (required, backup type)`

Available backup types:
- files; 
- incr_files;
- mysql;
- mysql_xtrabackup;
- mariadb_backup;
- postgresql;
- postgresql_basebackup;
- mongodb;
- redis;
- external.


2. Storage `-S [--storage-types] (optional, space-separated list of storages according to the pattern <storage_name> =<storage_type>)`,

Available remote storage types:
- s3;
- scp;
- sftp;
- smb;
- nfs;
- ftp;
- webdav.


3. Output `-O [--out-path] (optional, the path where the generated config should be saved)`

This will generate a configuration file for the job and output the details. 

For example, the next command will add an empty mysql backup job configuration file,
located by path '/etc/nxs-backup/conf.d/mysql.yml' and add two remote storage connection parameters
to the main config:
```sh
$ sudo nxs-backup generate -T mysql -S aws=s3 dumps=scp
nxs-backup: Successfully generated '/etc/nxs-backup/conf.d/mysql.yml' configuration file!
```

Instead of generating configuration files, you can use the empty configuration files available here.

The next step is to fill up the generated configuration files. For details go to [[Usage]]

**Testing of Conﬁguration**

You can verify that the configuration is correct by running nxs-backup with the -t option.

The program will process all configurations, display error messages if errors are found, and then exit:
```sh
$ sudo nxs-backup -t
The configuration is correct.
```

If the main configuration file is located on another path, you can define it with the optional
parameter `-c/--config (the path to the main configuration file)`.

To run nxs-backup on a regular schedule, you must add a call of nxs-backup to your **crontab** or **cron.d**.

This is an example of the cron file defining rules for Cron daemon to run nxs-backup by schedule.

/etc/cron.d/nxs-backup
```crontab
0 2 * * *       root    /usr/sbin/nxs-backup start all
```

