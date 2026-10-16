// SPDX-License-Identifier: MPL-2.0
// Resolve distribution WebKit helper paths inside the mounted AppImage.
// Keep WebKit's sandbox arguments, seccomp and system helper paths unchanged.
#define _GNU_SOURCE
#include <dlfcn.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

typedef void *(*spawn_fn)(void *, const char *const *, void **);
void *g_subprocess_launcher_spawnv(void *launcher, const char *const *argv, void **error) {
    spawn_fn original = (spawn_fn)dlsym(RTLD_NEXT, "g_subprocess_launcher_spawnv");
    if (!original) abort();
    const char *appdir = getenv("APPDIR");
    if (!appdir) return original(launcher, argv, error);
    const char *prefix = "/usr/lib/x86_64-linux-gnu/webkit2gtk-4.1/";
    size_t count = 0;
    while (argv[count]) count++;
    char **relocated = calloc(count + 1, sizeof(char *));
    if (!relocated) abort();
    for (size_t i = 0; i < count; i++) {
        const char *name = argv[i] + (strncmp(argv[i], prefix, strlen(prefix)) ? 0 : strlen(prefix));
        if (name != argv[i] && (!strcmp(name,"WebKitWebProcess") || !strcmp(name,"WebKitNetworkProcess") || !strcmp(name,"WebKitGPUProcess"))) {
            if (asprintf(&relocated[i], "%s%s%s", appdir, prefix, name) < 0) abort();
        } else {
            relocated[i] = strdup(argv[i]);
            if (!relocated[i]) abort();
        }
    }
    void *result = original(launcher, (const char *const *)relocated, error);
    for (size_t i = 0; i < count; i++) free(relocated[i]);
    free(relocated);
    return result;
}
