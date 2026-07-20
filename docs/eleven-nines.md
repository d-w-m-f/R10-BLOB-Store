# How AWS S3 achieves 99.999999999% durability?

One philosophy: Don't put all of your eggs in one basket.

It means: As soon as you receive data, spread it into multiple fisical devices replicated across multiple servers (when relevant, across multiple availability zones). Also, you should appl different strategies to ensure data corruption is as unlikely as possible.

## Strategies behind the Eleven Nines:

In a summarised phase, S3 implements a distributed erasure-coding safety net. 

In laymen's terms, you can say that s3 implements a variety of strategies at different levels - data level, storage level, geographical level - to ensure data integrity.

### 1. Data Replication:

When a customer uploads an object, S3 replicates it and spreads the copies across multiple storage devices, across multiple servers in at least 3 different availability zones (physically separated data centers).

### 2. Sharding and Data Integrity Checking:

When data is uploaded, it is broken into smaller pieces called shards. It reduces network overhead and failure rate while allowing for more robust strategies for both storage and recovery at the backend

When a file is uploaded, it gets:

1. Sharded into many pieces.
2. All of the shards get a checksum.
3. All of the shards get encoded using an error correction algorithm, which is called erasure coding.
4. After saving, the shards and parity shards are used to reconstruct the object, to verify the integrity of the upload and ensure retrievability.

#### Erasure Coding

How it works? 


### 3. Health Checkups

AWS constantly monitors the integrity of servers and storage devices. If one server or storage device is compromised, AWS will replace it with a new one.

Also, each disk is never full, its left with a little percentage of space empty, enough for handling emergencies. 


### 4. Bracketing

Before responding to the customer that an object has been successfully uploaded, S3 tries to reconstruct the object using the available shards and the parity shards. Only after it is certain that it will always be able to reconstruct your data, it considers the upload a success.


#### 5. File metadata

Like: filename, size, timestamps, permissions. Metadata is also replicated across multiple storage devices, servers and zones, to garantee that crucial information about an object is never lost.

#### 6. Humam error protection

For when someone ends up accidentaly deleting objects, S3 has a feature called versioning.

When something is deleted, S3 keeps a previous version of the object, until it is manually deleted.

Also, you are able to setup an object lock, that prevent objects from being deleted for a certain amount of time, even by users with admin access.



## What can I implement for R10?

### First, what is out of scope?

Obviously, I wont be able to implement something as robust as S3. I dont have for example the problem of storing petabytes of data, or having datacenters spread across the world.

R10 is a project for storage distributed at most inside one 'availability zone'. And your 'availability zone' is the hardware that you are running it.

So, features that are out of scope:

- Cross-availability zone replication
- Multi-region replication
- Cross-region replication
- Cross-account replication
- Cross-VPC replication
- Cross-AZ replication
- Cross-VPC replication

### Finally: What can I implement for R10?

1. Data replication: replicate data across multiple storage devices
2. Sharding and Data Integrity Checking: That can be implemented almost entirely
3. Health Checkups: Are a bit out of scope for now, that the project is small. In simulating hardware, so 'no machine failure' is to happen at that stage. But it is doable and important in the future.
4. Bracketing: I can implement bracketing for the storage devices. Although not as robust as AWS's bracketing, it will be enough for R10's needs.
5. File metadata: I can implement file metadata for the storage devices. Although not as robust as AWS's file metadata, it will be enough for R10's needs.
6. Humam error protection: I can implement humam error protection for the storage devices. Although not as robust as AWS's humam error protection, it will be enough for R10's needs.