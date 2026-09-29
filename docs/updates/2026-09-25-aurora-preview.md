# Aurora visual preview

An optional blue/violet glass-inspired finish for the dashboard, with softly rounded controls (10px) and panels (16px), existing navigation and all security controls. Classic retains its square corners. No OS dock, wallpaper assets, external fonts or third-party requests were added.

The **Style: Aurora / Style: Classic** button in the top toolbar changes only the visual finish, independently of Light / Dark / System. The choice is stored in this browser. The first preview defaults to Aurora; Classic restores the underlying styles without reloading.

The original CSS directory and theme script were copied to `backups/ui-before-glass-20260925/` before editing. The new appearance lives in `web/static/css/glass-preview.css`; original CSS files were not overwritten. To restore the pre-preview implementation entirely, restore `theme-modes.js` from that backup. Current layout fixes remain in both styles.
