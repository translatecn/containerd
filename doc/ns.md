```
stat -fc %T /sys/fs/cgroup/

对于cgroup v2，输出为cgroup2fs；对于cgroup v1，输出为tmpfs。
```



```
Permitted：进程所能使用的capabilities的上限集合，在该集合中有的权限，并不代表线程可以使用。必须要保证在Effective集合中有该权限。
Effective：有效的capabilities，这里的权限是Linux内核检查线程是否具有特权操作时检查的集合。
Inheritable：即继承。通过exec系统调用启动新进程时可以继承给新进程权限集合。注意，该权限集合继承给新进程后，也就是新进程的Permitted集合。
Bounding: Bounding限制了进程可以获得的集合，只有在Bounding集合中存在的权限，才能出现在Permitted和Inheritable集合中。
Ambient: Ambient集合中的权限会被应用到所有非特权进程上（特权进程，指当用户执行某一程序时，临时获得该程序所有者的身份）。
	然而，并不是所有在Ambient集合中的权限都会被保留，只有在Permitted和Effective集合中的权限，才会在被exec调用时保留。

在创建新的User namespace时不需要任何权限；而在创建其他类型的namespace（如UTS、PID、Mount、IPC、Network、Cgroupnamespace）时，
需要进程在对应User namespace中有CAP_SYS_ADMIN权限。

root@zjz:～# cat /proc/self/status | grep Cap
CapInh: 0000000000000000
CapPrm: 0000003fffffffff
CapEff: 0000003fffffffff
CapBnd: 0000003fffffffff
CapAmb: 0000000000000000

root@zjz:～# capsh --decode=0000003fffffffff



```


```
mount --make-shared /mntA      # 将挂载点设置为共享关系属性
mount --make-private /mntB     # 将挂载点设置为私有关系属性
mount --make-slave /mntC       # 将挂载点设置为从属关系属性
mount --make-unbindable /mntD  # 将挂载点设置为不可绑定关系属性
None：这种卷挂载将不会收到任何后续由宿主机(host)创建的在这个卷上或其子目录上的挂载。同样的，由容器创建的挂载在host上也是不可见的。这是默认的模式，等同于私有关系(MS_PRIVATE)。
HostToContainer：这种卷挂载将会收到之后所有的由宿主机(host)创建在该卷上或其子目录上的挂载，即宿主机在该卷内挂载的任何内容在容器中都是可见的，反过来，容器内挂载的内容在宿主机上是不可见的，即挂载传播是单向的，等同于从属关系(MS_SLAVE)。
Bidirectional：这种挂载机制和HostToContainer类似，即可以将宿主机上的挂载事件传播到容器内。此外，任何在容器中创建的挂载都会传播到宿主机，然后传播到使用相同卷的所有pod的所有容器，即挂载事件的传播是双向的，等同于共享关系(MS_SHARED)。



"CAP_CHOWN",
"CAP_DAC_OVERRIDE",
"CAP_FSETID",
"CAP_FOWNER",
"CAP_MKNOD",
"CAP_NET_RAW",
"CAP_SETGID",
"CAP_SETUID",
"CAP_SETFCAP",
"CAP_SETPCAP",
"CAP_NET_BIND_SERVICE",
"CAP_SYS_CHROOT",
"CAP_KILL",
"CAP_AUDIT_WRITE"
```




### 在介绍Time namespace前先介绍Linux系统调用中的几个时间类型。
- /proc/<pid>/timens_offsets
- CLOCK_REALTIME：操作系统对当前时间的展示（date展示的时间），随着系统time-of-day被修改而改变，例如用NTP(network time protocol)进行修改。
- CLOCK_MONOTONIC：单调时间，代表从过去某个固定的时间点开始的绝对的逝去时间，是不可被修改的。它不受任何系统time-of-day时钟修改的影响，如果想计算两个事件发生的间隔时间，它是最好的选择。
- CLOCK_BOOTTIME：系统启动时间（/proc/uptime中展示的时间）。
- CLOCK_BOOTTIME和CLOCK_MONOTONIC类似，也是单调的，在系统初始化时设定的基准数值是0。而且，不论系统是running还是suspend（这些都算是启动时间），CLOCK_BOOTTIME都会累积计时，直到系统reset或者shutdown。
```
# 打印当前的系统启动时间
root@zjz:～# uptime --pretty
up 4 days, 13 hours, 9 minutes
# 添加时间偏移量，7 天
root@zjz:～# unshare -T -- bash --norc
bash-5.0# echo "monotonic $((2*24*60*60)) 0" > /proc/$$/timens_offsets
bash-5.0# echo "boottime  $((7*24*60*60)) 0" > /proc/$$/timens_offsets
# 再次打印系统时间
bash-5.0# uptime --pretty
up 1 week, 4 days, 13 hours, 9 minutes
```