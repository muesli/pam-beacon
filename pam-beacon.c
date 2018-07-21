#include "string.h"
#include "stdlib.h"

#include "_cgo_export.h"

#define PAM_SM_AUTH
#define PAM_SM_PASSWORD
#include <security/pam_appl.h>
#include <security/pam_modules.h>

GoSlice argcvToSlice(int, const char**);

PAM_EXTERN int pam_sm_authenticate(pam_handle_t* pamh, int flags, int argc, const char** argv) {
  return goAuthenticate(pamh, flags, argcvToSlice(argc, argv));
}

PAM_EXTERN int pam_sm_setcred(pam_handle_t* pamh, int flags, int argc, const char** argv) {
  return setCred(pamh, flags, argcvToSlice(argc, argv));
}

GoSlice argcvToSlice(int argc, const char** argv) {
  GoString* strs = malloc(sizeof(GoString) * argc);

  GoSlice ret;
  ret.cap = argc;
  ret.len = argc;
  ret.data = (void*)strs;

  int i;
  for(i = 0; i < argc; i++) {
    strs[i] = *((GoString*)malloc(sizeof(GoString)));

    strs[i].p = (char*)argv[i];
    strs[i].n = strlen(argv[i]);
  }

  return ret;
}
