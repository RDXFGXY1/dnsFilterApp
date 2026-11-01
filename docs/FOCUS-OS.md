# Focus OS

Focus OS now lives in its own independent project folder: [side-apps/focus-os](../side-apps/focus-os/README.md).

Install the built `dist/side-apps/focus-os.adnf` through **Side Apps → Add app** in the main dashboard. Focus OS is no longer integrated into Discipline. The earlier experiment's database tables are left untouched so that existing data is not silently deleted, but this independent app does not consume them.

See [the Side Apps developer guide](SIDE-APPS.md) for the ADNF format, host API boundary, packaging, permissions, and isolation model.
