//go:build darwin && cgo

// Package posixprobe 仅由 Runtime 测试导入，验证原生 posix_spawn 不绕过隔离。
package posixprobe

/*
#include <errno.h>
#include <spawn.h>
#include <sys/wait.h>
extern char **environ;
static int harbor_spawn(int new_group) {
    pid_t pid;
    char *args[] = {"/usr/bin/true", NULL};
    posix_spawnattr_t attr;
    posix_spawnattr_init(&attr);
    if (new_group) {
        posix_spawnattr_setflags(&attr, POSIX_SPAWN_SETPGROUP);
        posix_spawnattr_setpgroup(&attr, 0);
    }
    int result = posix_spawn(&pid, args[0], NULL, &attr, args, environ);
    posix_spawnattr_destroy(&attr);
    if (!result) waitpid(pid, NULL, 0);
    return result;
}
*/
import "C"

func Spawn(newGroup bool) int {
	group := 0
	if newGroup {
		group = 1
	}
	return int(C.harbor_spawn(C.int(group)))
}
