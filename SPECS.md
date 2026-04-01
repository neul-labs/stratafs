This is a golang program that will do the following 
1. It will be looking at changes in a a set of directories 
2. The goal of the program is to have an updated .agentfs directory in all directories and subdirectories
3. This updated .agentfs will contain a sqlite database along with a usearch vector search index. These will contain parsed file chunks and means to vector search using usearch index. We will also enable fulltext search in sqlite as well for the chunks. 
4. This means when any file is updated we will update the sqlite database and usearch. We will first do softdeletes, and at some point we will run a compaction services to remove the entries. 
5. We will use fastembed to make the embeddings to be stored. 
6. We should assume that the job queue might need to persist and be run when the machine is not too busy
7. This will also expose a Model context protocol server along with a REST API as well. The key idea is to do recursive search for file chunks and the corresponding files using our structure. 
