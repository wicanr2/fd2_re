/*
 * FD2 Docker/Xvfb input helper.
 *
 * This helper deliberately injects only ordinary X11 key events.  It does
 * not change the game state, skip a scene, or call a remake API; it is used
 * by reproducible GUI traces when the test container has no xdotool.
 *
 * Usage:
 *   fd2-xtest-input Escape:1000 Down:1000 Shift+F5:1500 Ctrl+F6:1500
 *
 * The number is the delay, in milliseconds, after releasing the key.  A key
 * is pressed and released as two synchronised XTest events so Ebiten can
 * observe a real just-pressed edge.  The helper waits at most 30 seconds for
 * a visible game-sized window; it never falls back to sending keys to root.
 */

#include <X11/Xlib.h>
#include <X11/Xatom.h>
#include <X11/keysym.h>
#include <X11/extensions/XTest.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

static Window choose_window(Display *dpy, Window root) {
    Window chosen = root;
    unsigned best_area = 0;
    Window parent = 0, *children = NULL;
    unsigned nchildren = 0;
    if (!XQueryTree(dpy, root, &root, &parent, &children, &nchildren)) {
        return root;
    }
    for (unsigned i = 0; i < nchildren; ++i) {
        XWindowAttributes attrs;
        if (!XGetWindowAttributes(dpy, children[i], &attrs) ||
            attrs.map_state != IsViewable) {
            continue;
        }
        unsigned area = (unsigned)(attrs.width > 0 ? attrs.width : 0) *
                        (unsigned)(attrs.height > 0 ? attrs.height : 0);
        fprintf(stderr, "[fd2-xtest] window=0x%lx size=%dx%d map=%d\n",
                (unsigned long)children[i], attrs.width, attrs.height,
                attrs.map_state);
        if (area > best_area && attrs.width >= 320 && attrs.height >= 200) {
            chosen = children[i];
            best_area = area;
        }
        Window nested = choose_window(dpy, children[i]);
        if (nested != children[i]) {
            XWindowAttributes nested_attrs;
            if (XGetWindowAttributes(dpy, nested, &nested_attrs) &&
                nested_attrs.map_state == IsViewable) {
                unsigned nested_area =
                    (unsigned)(nested_attrs.width > 0 ? nested_attrs.width : 0) *
                    (unsigned)(nested_attrs.height > 0 ? nested_attrs.height : 0);
                if (nested_area > best_area && nested_attrs.width >= 320 &&
                    nested_attrs.height >= 200) {
                    chosen = nested;
                    best_area = nested_area;
                }
            }
        }
    }
    if (children) {
        XFree(children);
    }
    return chosen;
}

static KeySym key_symbol(const char *name) {
    if (!strcmp(name, "Escape")) return XK_Escape;
    if (!strcmp(name, "Return") || !strcmp(name, "Enter")) return XK_Return;
    if (!strcmp(name, "Space")) return XK_space;
    if (!strcmp(name, "Up")) return XK_Up;
    if (!strcmp(name, "Down")) return XK_Down;
    if (!strcmp(name, "Left")) return XK_Left;
    if (!strcmp(name, "Right")) return XK_Right;
    if (!strcmp(name, "F1")) return XK_F1;
    if (!strcmp(name, "F2")) return XK_F2;
    if (!strcmp(name, "F3")) return XK_F3;
    if (!strcmp(name, "F4")) return XK_F4;
    if (!strcmp(name, "F5")) return XK_F5;
    if (!strcmp(name, "F6")) return XK_F6;
    if (!strcmp(name, "F7")) return XK_F7;
    if (!strcmp(name, "F8")) return XK_F8;
    if (!strcmp(name, "F9")) return XK_F9;
    if (!strcmp(name, "F10")) return XK_F10;
    return XStringToKeysym(name);
}

int main(int argc, char **argv) {
    if (argc < 2) {
        fprintf(stderr, "usage: %s Key:delay_ms [... ]\n", argv[0]);
        return 2;
    }
    Display *dpy = XOpenDisplay(NULL);
    if (!dpy) {
        fprintf(stderr, "cannot open DISPLAY\n");
        return 3;
    }
    Window root = DefaultRootWindow(dpy);
    Window target = root;
    for (int attempt = 0; attempt < 300 && target == root; ++attempt) {
        target = choose_window(dpy, root);
        if (target == root) {
            usleep(100000);
        }
    }
    if (target == root) {
        fprintf(stderr, "no visible game-sized window within 30 seconds\n");
        XCloseDisplay(dpy);
        return 7;
    }
    fprintf(stderr, "[fd2-xtest] target=0x%lx\n", (unsigned long)target);
    XMapRaised(dpy, target);
    XSetInputFocus(dpy, target, RevertToParent, CurrentTime);
    XTestGrabControl(dpy, True);
    XSync(dpy, False);

    for (int i = 1; i < argc; ++i) {
        char *spec = strdup(argv[i]);
        if (!spec) return 4;
        char *colon = strrchr(spec, ':');
        int delay_ms = 200;
        if (colon) {
            *colon = '\0';
            delay_ms = atoi(colon + 1);
            if (delay_ms < 0) delay_ms = 0;
        }
        KeySym modifier_sym = NoSymbol;
        const char *modifier_name = NULL;
        if (!strncmp(spec, "Shift+", 6)) {
            modifier_sym = XK_Shift_L;
            modifier_name = "Shift";
            memmove(spec, spec + 6, strlen(spec + 6) + 1);
        } else if (!strncmp(spec, "Ctrl+", 5)) {
            modifier_sym = XK_Control_L;
            modifier_name = "Ctrl";
            memmove(spec, spec + 5, strlen(spec + 5) + 1);
        } else if (!strncmp(spec, "Alt+", 4)) {
            modifier_sym = XK_Alt_L;
            modifier_name = "Alt";
            memmove(spec, spec + 4, strlen(spec + 4) + 1);
        }
        KeySym sym = key_symbol(spec);
        free(spec);
        if (sym == NoSymbol) {
            fprintf(stderr, "unknown key in %s\n", argv[i]);
            XCloseDisplay(dpy);
            return 5;
        }
        KeyCode code = XKeysymToKeycode(dpy, sym);
        if (!code) {
            fprintf(stderr, "no keycode for %s\n", argv[i]);
            XCloseDisplay(dpy);
            return 6;
        }
        KeyCode modifier_code = modifier_sym == NoSymbol ? 0 :
                                XKeysymToKeycode(dpy, modifier_sym);
        if (modifier_sym != NoSymbol && !modifier_code) {
            fprintf(stderr, "no keycode for %s in %s\n", modifier_name, argv[i]);
            XCloseDisplay(dpy);
            return 8;
        }
        if (modifier_code) {
            XTestFakeKeyEvent(dpy, modifier_code, True, CurrentTime);
            XSync(dpy, False);
            /* Match an ordinary chord: let the game observe the held
             * modifier before the function-key just-pressed edge. */
            usleep(120000);
        }
        XTestFakeKeyEvent(dpy, code, True, CurrentTime);
        XSync(dpy, False);
        usleep(60000);
        XTestFakeKeyEvent(dpy, code, False, CurrentTime);
        if (modifier_code) {
            XTestFakeKeyEvent(dpy, modifier_code, False, CurrentTime);
        }
        XSync(dpy, False);
        usleep((useconds_t)delay_ms * 1000U);
    }
    XTestGrabControl(dpy, False);
    XCloseDisplay(dpy);
    return 0;
}
