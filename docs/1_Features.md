## 1. Features ##

Here is a list of the main features that nxs-backup provides.
- Full data backup
    - Files backups:
        * Discrete
        * Incremental
    - Database backups:
        * Discrete
        * Logical backups of MySQL/Percona (5.7/8.0/all versions)
        * Logical backups of MariaDB (10/11/all versions)
        * Physical backups by Xtrabackup (2.4/8.0) of MySQL/Percona (5.7/8.0/all versions)
        * Physical backups by MariaDB-backup of MariaDB (10/11/all versions)
        * Logical backups of PostgreSQL (9/10/11/12/13/14/15/16/all versions)
        * Physical backups by Basebackups of PostgreSQL (9/10/11/12/13/14/15/16/all versions)
        * Backups of MongoDB (4.0/4.2/4.4/5.0/6.0/7.0/all versions)
        * Backups of Redis (all versions)
    - Support of user-defined scripts that extend functionality
- Upload and manage backups to the remote storage:
    * S3 (Simple Storage Service that provides object storage through a web interface. Supported by clouds e.g. AWS, GCP)
    * SSH (SFTP)
    * FTP
    * CIFS (SMB)
    * NFS
    * WebDAV
- Notifications via email and webhooks about events of the backup process;
- Collect, export, and save metrics in Prometheus-compatible format;
- Limiting resource consumption:
    * CPU usage
    * disk rate
    * remote storage

