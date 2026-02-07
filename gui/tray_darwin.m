#import <Cocoa/Cocoa.h>

extern void tray_callback_show();
extern void tray_callback_quit();
extern void tray_callback_about();

@interface SloPNTrayDelegate : NSObject
- (void)onShow:(id)sender;
- (void)onQuit:(id)sender;
- (void)onAbout:(id)sender;
@end

@implementation SloPNTrayDelegate
- (void)onShow:(id)sender {
    tray_callback_show();
}
- (void)onQuit:(id)sender {
    tray_callback_quit();
}
- (void)onAbout:(id)sender {
    tray_callback_about();
}
@end

static SloPNTrayDelegate *delegate;
static NSStatusItem *statusItem;

void init_tray(const char* title) {
    NSLog(@"SloPN: init_tray called with title: %s", title);
    dispatch_async(dispatch_get_main_queue(), ^{
        NSLog(@"SloPN: Running on main thread, creating status item...");
        
        if (delegate == nil) {
            delegate = [[SloPNTrayDelegate alloc] init];
        }
        
        statusItem = [[NSStatusBar systemStatusBar] statusItemWithLength:NSSquareStatusItemLength];
        [statusItem retain];
        
        if (statusItem == nil) {
            NSLog(@"SloPN: ERROR - Could not create statusItem!");
            return;
        }

        if (@available(macOS 11.0, *)) {
            NSImageSymbolConfiguration *config = [NSImageSymbolConfiguration configurationWithScale:NSImageSymbolScaleLarge];
            NSImage *image = [NSImage imageWithSystemSymbolName:@"shield" accessibilityDescription:@"SloPN"];
            statusItem.button.image = [image imageWithSymbolConfiguration:config];
        }
        
        NSMenu *menu = [[NSMenu alloc] init];
        [menu addItemWithTitle:@"Show Dashboard" action:@selector(onShow:) keyEquivalent:@""];
        [menu addItem:[NSMenuItem separatorItem]];
        [menu addItemWithTitle:@"About SloPN" action:@selector(onAbout:) keyEquivalent:@""];
        [menu addItem:[NSMenuItem separatorItem]];
        [menu addItemWithTitle:@"Quit" action:@selector(onQuit:) keyEquivalent:@"q"];
        
        for (NSMenuItem *item in menu.itemArray) {
            [item setTarget:delegate];
        }
        
        [statusItem setMenu:menu];
        [statusItem setVisible:YES];
        NSLog(@"SloPN: Tray initialization complete.");
    });
}

void update_tray_status(int connected) {
    dispatch_async(dispatch_get_main_queue(), ^{
        if (statusItem == nil) return;
        
        if (@available(macOS 11.0, *)) {
            NSString *name = connected ? @"shield.fill" : @"shield";
            NSImageSymbolConfiguration *config;
            
            if (connected) {
                // Use hierarchical green coloring
                if (@available(macOS 12.0, *)) {
                    config = [NSImageSymbolConfiguration configurationWithPaletteColors:@[[NSColor systemGreenColor]]];
                } else {
                    config = [NSImageSymbolConfiguration configurationWithHierarchicalColor:[NSColor systemGreenColor]];
                }
            } else {
                config = [NSImageSymbolConfiguration configurationWithScale:NSImageSymbolScaleLarge];
            }
            
            // Merge with Large scale if possible
            if (@available(macOS 12.0, *)) {
                NSImageSymbolConfiguration *scaleConfig = [NSImageSymbolConfiguration configurationWithScale:NSImageSymbolScaleLarge];
                config = [config configurationByApplyingConfiguration:scaleConfig];
            }

            NSImage *image = [NSImage imageWithSystemSymbolName:name accessibilityDescription:@"SloPN"];
            [statusItem.button setImage:[image imageWithSymbolConfiguration:config]];
            
            // Clear tint to let the symbol color shine through
            statusItem.button.contentTintColor = nil;
        }
    });
}
