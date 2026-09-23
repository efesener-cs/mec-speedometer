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

static void find_window_for_pid(Display *d, Window w, Atom pid_atom, pid_t pid,
		int *best_area, int *best_x, int *best_y, int *best_width, int *best_height) {
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
		if (XGetWindowAttributes(d, w, &a) && a.map_state == IsViewable && a.width >= 160 && a.height >= 90 &&
			XTranslateCoordinates(d, w, DefaultRootWindow(d), 0, 0, &rx, &ry, &child)) {
			int area = a.width * a.height;
			if (area > *best_area) {
				*best_area = area;
				*best_x = rx; *best_y = ry; *best_width = a.width; *best_height = a.height;
			}
		}
	} else if (data != NULL) {
		XFree(data);
	}
	Window root, parent, *children = NULL;
	unsigned int count = 0;
	if (!XQueryTree(d, w, &root, &parent, &children, &count)) return;
	for (unsigned int i = 0; i < count; i++) {
		find_window_for_pid(d, children[i], pid_atom, pid, best_area, best_x, best_y, best_width, best_height);
	}
	if (children) XFree(children);
}

static int window_geometry(Display *d, Window w, int *x, int *y, int *width, int *height) {
	XWindowAttributes a;
	int rx, ry;
	Window child;
	if (!XGetWindowAttributes(d, w, &a) || a.map_state != IsViewable || a.width < 160 || a.height < 90) return 0;
	if (!XTranslateCoordinates(d, w, DefaultRootWindow(d), 0, 0, &rx, &ry, &child)) return 0;
	*x = rx; *y = ry; *width = a.width; *height = a.height;
	return 1;
}

static void *mec_overlay_new(void) {
	// Prefer XWayland when both the Wayland and X11 backends are available.
	gdk_set_allowed_backends("x11");
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
	gtk_window_set_position(GTK_WINDOW(o->window), GTK_WIN_POS_CENTER);
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
	cairo_region_t *click_through = cairo_region_create();
	gtk_widget_input_shape_combine_region(o->window, click_through);
	cairo_region_destroy(click_through);
	return o;
}

static void mec_overlay_update(void *handle, int pid, const char *text) {
	MECOverlay *o = (MECOverlay *)handle;
	if (!o) return;
	gtk_label_set_text(GTK_LABEL(o->label), text);
	if (pid > 0) {
		int x, y, w, h;
		Atom pid_atom = XInternAtom(o->display, "_NET_WM_PID", False);
		int best_area = 0;
		if (pid_atom != None) {
			find_window_for_pid(o->display, DefaultRootWindow(o->display), pid_atom, pid,
				&best_area, &x, &y, &w, &h);
		}
		if (best_area > 0) {
			gtk_window_move(GTK_WINDOW(o->window), x + (w - 280) / 2, y + h / 10);
		} else {
			// Proton/game-scope may publish the wrapper PID; follow the active game window.
			Atom active_atom = XInternAtom(o->display, "_NET_ACTIVE_WINDOW", True);
			Atom actual;
			int format;
			unsigned long count, after;
			unsigned char *data = NULL;
			if (active_atom != None && XGetWindowProperty(o->display, DefaultRootWindow(o->display),
				active_atom, 0, 1, False, XA_WINDOW, &actual, &format, &count, &after, &data) == Success &&
				data != NULL && count == 1) {
				Window active = *(Window *)data;
				if (window_geometry(o->display, active, &x, &y, &w, &h)) {
					gtk_window_move(GTK_WINDOW(o->window), x + (w - 280) / 2, y + h / 10);
				}
			}
			if (data) XFree(data);
		}
	}
	GdkWindow *gdk_window = gtk_widget_get_window(o->window);
	if (gdk_window) XRaiseWindow(o->display, GDK_WINDOW_XID(gdk_window));
	XFlush(o->display);
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
