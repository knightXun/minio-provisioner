# Copyright 2017 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Modified from https://github.com/rootfs/nfs-ganesha-docker by Huamin Chen
FROM centos


RUN yum install -y  net-tools file

ADD s3cmd-2.0.2-1.el7.noarch.rpm /root

ADD python-magic-5.11-35.el7.noarch.rpm /root

ADD python-dateutil-1.5-7.el7.noarch.rpm /root

RUN rpm -ivh /root/python-magic-5.11-35.el7.noarch.rpm && rpm -ivh /root/python-dateutil-1.5-7.el7.noarch.rpm && rpm -ivh /root/s3cmd-2.0.2-1.el7.noarch.rpm

COPY minio-provisioner /minio-provisioner
ENTRYPOINT ["/minio-provisioner"]
