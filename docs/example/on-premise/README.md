# On-premise configuration files example

Here is example of configs for create backups of files and databases for projects located on-premise in
directory `/var/www` with exclude of `bitrix` files and upload backups to remote s3 and ssh storages.

Main config file located at [/etc/nxs-backup/nxs-backup.yml](nxs-backup.yml)

Files backup job config at [/etc/nxs-backup/conf.d/files.yml](conf.d/files_s3.yml)

Mysql database backup job config at [/etc/nxs-backup/conf.d/mysql.yml](conf.d/mysql_scp.yml)

PSQL database backup job config at [/etc/nxs-backup/conf.d/psql.yml](conf.d/psql_scp.yml)
