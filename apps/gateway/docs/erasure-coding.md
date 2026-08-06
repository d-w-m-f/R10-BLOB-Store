# How erasure coding works at R10 Blob Store

First, we break the uploaded file into N chunks. 

Then, we create M parity chunks using reed-solomon encoding algorithm..

So, we are left with (N + M) = 3/2 * N chunks in total. To rebuild the file, we need to retrieve at least N chunks (data + parity chunks) from the storage system. 

The space complexity of this process is 3/2 * N chunks in total, with constant 1.5 times the original file size in storage space spent

## Blocks, sizes, erasure strategies and more

### Chunks

Chunk Size will be 32MB.

### Erasure coding strategies

There are gonna utilize 8+4 erasure coding strategy, paired w/ reed-solomon encoding algorithm.  
 

### Dealing with archives

#### Case 1:  Archives that are less than 128KB 

Those are saved inline, in 'inline-machines', to not raise complexity on orchestrating inline and block inside every machine.

#### Case 2: Archives between 128KB and 32MB

Those are grouped inside remaining block spaces, as a single block (1 chunk), inside a worker machine, without erasure coding applied. We will need a data structure to keep track of blocks with remaining space, and how much space is remaining in each one.

#### Case 3: Archives > 32MB

Those are chunkerized, full chunks are then processed into 8+4 erasure coding. The remaining chunk is treated as a case 2.


### Machines

We will utilize 5 machines. That not a hard limit, is just to orient ourselves for now. The number will be a variable in the system, and all code and engeneering must account to that.

Machines 1-4 are block machines, meaning they are used for storing blocks of data + parity chunks

Machine 5 is an inline machine, meaning it is used for storing inline data

Key takeaway, machines are split in two types: block and inline.
NOTE: That must be reflected in the database schemas, and in the code.  

## Challanges

1. We need to define a data structure to keep track of blocks with remaining space, and how much space is remaining in each one.

2. We need to deal with reprocessing of chunk encoding. It cant be that a case 2 processing triggers more than one re-encoding of a block.

3. We need to deal with locks on processing chunks, to avoid race conditions on chunk processing ot actions that can lead to inconsistent state.

4. We need to make machine discovery and remaining space.