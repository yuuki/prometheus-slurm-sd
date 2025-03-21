FROM rockylinux:9

# Set Slurm version
ENV SLURM_VERSION=22.05.9
ENV SLURM_DOWNLOAD_URL=https://download.schedmd.com/slurm/slurm-${SLURM_VERSION}.tar.bz2

# Install dependencies
RUN dnf -y update && dnf -y install \
    epel-release \
    && dnf config-manager --set-enabled crb \
    && dnf -y install \
    gcc-c++ \
    make \
    bzip2 \
    ca-certificates \
    git \
    gnupg2 \
    dbus-devel \
    libcurl-devel \
    hdf5-devel \
    json-c-devel \
    munge-devel \
    munge-libs \
    pam-devel \
    openssl-devel \
    munge \
    python3 \
    python3-devel \
    python3-pip \
    wget \
    file \
    readline-devel \
    && dnf clean all

# Download and extract Slurm
WORKDIR /tmp
RUN curl -o slurm.tar.bz2 "${SLURM_DOWNLOAD_URL}" \
    && tar -xf slurm.tar.bz2 \
    && rm slurm.tar.bz2 \
    && cd slurm-${SLURM_VERSION} \
    && ./configure --prefix=/usr --sysconfdir=/etc/slurm \
       --with-munge=/usr \
    && make -j$(nproc) \
    && make install \
    && cd .. \
    && rm -rf slurm-${SLURM_VERSION}

# Create required directories
RUN mkdir -p /etc/slurm /var/log/slurm /var/run/slurm /var/lib/slurm /var/spool/slurmd /var/spool/slurmctld \
    && chmod -R 755 /var/log/slurm /var/run/slurm /var/lib/slurm /var/spool/slurmd /var/spool/slurmctld

# Create empty slurm.conf to avoid warnings
RUN touch /etc/slurm/slurm.conf

ARG GOSU_VERSION=1.17

RUN set -ex \
    && wget -O /usr/local/bin/gosu "https://github.com/tianon/gosu/releases/download/$GOSU_VERSION/gosu-amd64" \
    && wget -O /usr/local/bin/gosu.asc "https://github.com/tianon/gosu/releases/download/$GOSU_VERSION/gosu-amd64.asc" \
    && export GNUPGHOME="$(mktemp -d)" \
    && gpg --batch --keyserver hkps://keys.openpgp.org --recv-keys B42F6819007F00F88E364FD4036A9C25BF357DD4 \
    && gpg --batch --verify /usr/local/bin/gosu.asc /usr/local/bin/gosu \
    && rm -rf "${GNUPGHOME}" /usr/local/bin/gosu.asc \
    && chmod +x /usr/local/bin/gosu \
    && gosu nobody true

RUN /sbin/create-munge-key

RUN dnf -y update && dnf -y install \
    dbus \
    dbus-daemon \
    && dnf clean all

# Copy entrypoint script
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Expose ports
# slurmctld
EXPOSE 6817
# slurmd
EXPOSE 6818
# slurmrestd
EXPOSE 6820

# Set entrypoint
ENTRYPOINT ["/entrypoint.sh"]

# Default command is to print help
CMD ["--help"]
