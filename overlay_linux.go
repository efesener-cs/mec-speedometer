//go:build linux && cgo

package main

/*
#cgo pkg-config: gtk+-3.0 x11
#include <gtk/gtk.h>
#include <gdk/gdkx.h>
#include <X11/Xatom.h>
#include <X11/Xlib.h>
#include <stdint.h>
#include <stdlib.h>
#include <sys/types.h>

typedef struct {
	GtkWidget *window;
	GtkWidget *label;
	Display *display;
} MECOverlay;

static gboolean draw_background(GtkWidget *widget, cairo_t *cr, gpointer data) {
	GtkAllocation a;
	gtk_widget_get_allocation(widget, &a);
	double radius = 14.0;
	cairo_new_sub_path(cr);
	cairo_arc(cr, a.width-radius, radius, radius, -1.5708, 0.0);
	cairo_arc(cr, a.width-radius, a.height-radius, radius, 0.0, 1.5708);
	cairo_arc(cr, radius, a.height-radius, radius, 1.5708, 3.14159);
	cairo_arc(cr, radius, radius, radius, 3.14159, 4.71239);
	cairo_close_path(cr);
	cairo_set_source_rgba(cr, 0.02, 0.04, 0.05, 0.72);
	cairo_fill(cr);
	return FALSE;
}

static int find_window_for_pid(Display *d, Window w, Atom pid_atom, pid_t pid,
		int *x, int *y, int *width, int *height) {
	Atom actual;
	int format;
	unsigned long nitems, after;
	unsigned char *data = NULL;
	if (XGetWindowProperty(d, w, pid_atom, 0, 1, False, XA_CARDINAL,
			&actual, &format, &nitems, &after, &data) == Success &&
			data != NULL && nitems == 1 && *(unsigned long *)data == (unsigned long)pid) {
		XFree(data);
		XWindowAttributes a;
	int rx, ry;
	Window child;
		if (XGetWindowAttributes(d, w, &a) && a.map_state == IsViewable &&
			XTranslateCoordinates(d, w, DefaultRootWindow(d), 0, 0, &rx, &ry, &child)) {
			*x = rx; *y = ry; *width = a.width; *height = a.height;
			return 1;
		}
	} else if (data != NULL) {
		XFree(data);
	}
	Window root, parent, *children = NULL;
	unsigned int count = 0;
	if (!XQueryTree(d, w, &root, &parent, &children, &count)) return 0;
	for (unsigned int i = 0; i < count; i++) {
		if (find_window_for_pid(d, children[i], pid_atom, pid, x, y, width, height)) {
			XFree(children);
			return 1;
		}
	}
	if (children) XFree(children);
	return 0;
}

static void *mec_overlay_new(void) {
	if (!gtk_init_check(NULL, NULL)) return NULL;
	GdkDisplay *gdk = gdk_display_get_default();
	if (!gdk || !GDK_IS_X11_DISPLAY(gdk)) return NULL;
	MECOverlay *o = g_new0(MECOverlay, 1);
	GdkScreen *screen = gdk_screen_get_default();
	GdkVisual *visual = gdk_screen_get_rgba_visual(screen);
	o->window = gtk_window_new(GTK_WINDOW_TOPLEVEL);
	if (visual) gtk_widget_set_visual(o->window, visual);
	gtk_window_set_decorated(GTK_WINDOW(o->window), FALSE);
	gtk_window_set_resizable(GTK_WINDOW(o->window), FALSE);
	gtk_window_set_keep_above(GTK_WINDOW(o->window), TRUE);
	gtk_window_set_skip_taskbar_hint(GTK_WINDOW(o->window), TRUE);
	gtk_window_set_skip_pager_hint(GTK_WINDOW(o->window), TRUE);
	gtk_window_set_accept_focus(GTK_WINDOW(o->window), FALSE);
	gtk_window_set_focus_on_map(GTK_WINDOW(o->window), FALSE);
	gtk_widget_set_app_paintable(o->window, TRUE);
	gtk_widget_set_size_request(o->window, 280, 82);
	g_signal_connect(o->window, "draw", G_CALLBACK(draw_background), NULL);
	o->label = gtk_label_new("Waiting for game...");
	GtkWidget *box = gtk_box_new(GTK_ORIENTATION_VERTICAL, 0);
	gtk_widget_set_margin_start(box, 18);
	gtk_widget_set_margin_end(box, 18);
	gtk_widget_set_margin_top(box, 12);
	gtk_widget_set_margin_bottom(box, 12);
	gtk_widget_set_halign(o->label, GTK_ALIGN_CENTER);
	gtk_widget_set_valign(o->label, GTK_ALIGN_CENTER);
	GtkCssProvider *css = gtk_css_provider_new();
	gtk_css_provider_load_from_data(css, "label { color: #ffffff; font: bold 38px sans-serif; }", -1, NULL);
	gtk_style_context_add_provider(gtk_widget_get_style_context(o->label), GTK_STYLE_PROVIDER(css), GTK_STYLE_PROVIDER_PRIORITY_APPLICATION);
	g_object_unref(css);
	gtk_container_add(GTK_CONTAINER(o->window), box);
	gtk_container_add(GTK_CONTAINER(box), o->label);
	o->display = GDK_DISPLAY_XDISPLAY(gdk);
	gtk_widget_show_all(o->window);
	return o;
}

static void mec_overlay_update(void *handle, int pid, const char *text) {
	MECOverlay *o = (MECOverlay *)handle;
	if (!o) return;
	gtk_label_set_text(GTK_LABEL(o->label), text);
	if (pid > 0) {
		int x, y, w, h;
		Atom pid_atom = XInternAtom(o->display, "_NET_WM_PID", True);
		if (pid_atom != None && find_window_for_pid(o->display, DefaultRootWindow(o->display), pid_atom, pid, &x, &y, &w, &h)) {
			gtk_window_move(GTK_WINDOW(o->window), x + (w - 280) / 2, y + h / 10);
		}
	}
	while (gtk_events_pending()) gtk_main_iteration();
}

static void mec_overlay_close(void *handle) {
	MECOverlay *o = (MECOverlay *)handle;
	if (!o) return;
	gtk_widget_destroy(o->window);
	while (gtk_events_pending()) gtk_main_iteration();
	g_free(o);
}
*/
import "C"

import (
	"runtime"
	"unsafe"
)

type speedOverlay struct{ handle unsafe.Pointer }

func newSpeedOverlay() *speedOverlay {
	runtime.LockOSThread()
	p := C.mec_overlay_new()
	if p == nil {
		runtime.UnlockOSThread()
		return nil
	}
	return &speedOverlay{handle: p}
}

func (o *speedOverlay) update(pid int, text string) {
	cText := C.CString(text)
	defer C.free(unsafe.Pointer(cText))
	C.mec_overlay_update(o.handle, C.int(pid), cText)
}

func (o *speedOverlay) close() {
	if o == nil || o.handle == nil {
		return
	}
	C.mec_overlay_close(o.handle)
	o.handle = nil
	runtime.UnlockOSThread()
}
