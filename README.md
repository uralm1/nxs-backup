# nxs-backup (fork)

**Nxs-backup** is a tool for creating and delivery backups, rotating it locally and on remote storages, compatible with
GNU/Linux distributions.

## Introduction

> [!NOTE]
> Some configuration options have been **changed** including job types, so consider checking the updated [documentation pages](https://github.com/uralm1/nxs-backup/tree/my/docs).

### Features

- Full data backup
  - File backups:
    - Discrete files backups
    - Incremental files backups
  - Database backups:
    - Logical backups of MySQL/Percona (5.7/8.0/_all versions_)
    - Logical backups of MariaDB (10/11/_all versions_)
    - Physical backups by Xtrabackup (2.4/8.0) of MySQL/Percona (5.7/8.0/_all versions_)
    - Physical backups by MariaDB-backup of MariaDB (10/11/_all versions_)
    - Logical backups of PostgreSQL (9/10/11/12/13/14/15/16/_all versions_)
    - Physical backups by Basebackups of PostgreSQL (9/10/11/12/13/14/15/16/_all versions_)
    - Backups of MongoDB (4.0/4.2/4.4/5.0/6.0/7.0/_all versions_)
    - Backups of Redis (_all versions_)
  - Support of user-defined scripts that extend functionality
- Upload and manage backups to the remote storages:
  - S3 (Simple Storage Service that provides object storage through a web interface. Supported by clouds e.g. AWS, GCP)
  - SSH (SFTP)
  - FTP
  - CIFS (SMB)
  - NFS
  - WebDAV
- Fine-tune the database backup process with additional options for optimization purposes
- Notifications about events of the backup process via email and webhooks
- Collect, export, and save metrics in Prometheus-compatible format
- Limiting resource consumption:
  - CPU usage
  - local disk rate
  - remote storage rate

## Quickstart

- Clone the repo
  ```sh
  git clone https://github.com/uralm1/nxs-backup.git
  ```
  
### On-premise (bare-metal or virtual machine)

- Install nxs-backup, just download and unpack archive for your CPU architecture.
  ```sh
  curl -L https://github.com/uralm1/nxs-backup/releases/latest/download/ural-nxs-backup-amd64.tar.gz -o /tmp/nxs-backup.tar.gz
  tar xf /tmp/nxs-backup.tar.gz -C /tmp
  sudo mv /tmp/nxs-backup /usr/sbin/nxs-backup
  sudo chown root:root /usr/sbin/nxs-backup
  ```
> [!NOTE]
> If you need specific version of nxs-backup, or different architecture, you can find it on [release page](https://github.com/uralm1/nxs-backup/releases).
- Check that installation successful:
  ```sh
  sudo nxs-backup --version
  ```
- Generate configuration files like described [here](https://github.com/uralm1/nxs-backup/tree/my/docs/2_Configuration.md) or update
  provided `nxs-backup.conf` and jobs configs in `cond.d` dir with your parameters (see [docs](https://github.com/uralm1/nxs-backup/tree/my/docs/3.1_Usage.md) for details)
- For starting nxs-backup process run:
  ```sh
  sudo nxs-backup start
  ```

### Docker-compose

- Go to docker compose directory
  ```sh
  cd nxs-backup/.deploy/docker-compose/
  ```
- Update `nxs-backup.conf` file with your parameters (see [docs](https://github.com/uralm1/nxs-backup/tree/my/docs/3.1_Usage.md) for details)
- Launch the nxs-backup with command:
  ```sh
  docker compose up -d --pull
  ```

### Kubernetes

- Go to kubernetes directory
  ```sh
  cd nxs-backup/.deploy/kubernetes/
  ```
- Install [nxs-universal-chart](https://github.com/nixys/nxs-universal-chart) (`Helm 3` is required):
  ```sh
  helm repo add nixys https://registry.nixys.io/chartrepo/public
  ```
- Find examples of `helm values` [here](/docs/example/kubernetes/README.md)
- Fill up your `values.yaml` with correct nxs-backup [configs](https://github.com/uralm1/nxs-backup/tree/my/docs/3.1_Usage.md)
- Launch nxs-backup with command:
  ```sh
  helm -n $NAMESPACE_SERVICE_NAME install nxs-backup nixys/nxs-universal-chart -f values.yaml
  ```
  where $NAMESPACE_SERVICE_NAME is the namespace in which to back up your data

## License

nxs-backup is released under the [Apache-2.0 license](LICENSE).

[tg-news-url]: https://t.me/nxs_backup
[tg-chat-url]: https://t.me/nxs_backup_chat
