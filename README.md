# cache
A simple local cache

1. The number of slices is determined by the size of the number of keys. There is no lock competition for reads and writes between slices; locks exist only for reads and writes within the same slice.

2. The expiration time of keys is periodically checked for deletion by a time wheel

3. Cached base datatypes can be synchronized to a file, or loaded from a file

一个简单的本地缓存

1. 分片的数量由key数量的大小决定。分片之间的读写不存在锁的竞争，锁只存在于同一分片内的读写。

2. 通过时间轮定期检查key的过期时间进行删除。

3. 缓存的基础数据类型可以同步到一个文件，或者从文件中加载。