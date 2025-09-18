# Stash: Reforged

This is changed version of Stash App (from 0.29-dev)!\
I changed original repository for my own personal requests so it could be better than original app (but it harder to run server and maybe a little bit unstable).

_**If you want you can suggest some improvements or functionality - you are welcome**_! (Maybe this functionality will appear faster here than original stash app).

This version will be always in develop branch.

### Some new features

- [x] **Trimmed segments** - allows you to skip start and end of the video with intro and advertise (without trim original video)
- [x] **Scenes recommendations** (by weights of tags, performers, studio and groups) - shows you suggested content to watch!
- [x] Setting for show/hide percent of similarity in suggested scenes
- [x] Status for ignore tag for suggested scenes (these tags will not change percent of similarity)
- [x] **Pose tags block** - shows more informative tags to select (for tags with pose flag)
- [x] **"Random" button, "Random best" button** - shows random scene by rating
- [x] **Converter broken formats to mp4 and HLS to mp4** - for real converting original video to valid format (like transcode but replaces original video). Convertable video will copy to temp folder (.stash/temp) before file will convert and save correctly (for safe)
- [x] **Tag colors** - mark important tags (color presets and palette of already used colors make color edit more comfortable)
- [x] **O-Count stats and graphics**
- [x] **Multiple images for performers** (gallery carousel with set some image as primary) - you can now add 2-3 the best photos!
- [x] Auto-detect potentially broken videos (non-mp4 formats and HLS mp4)
- [x] Tag weights - some tags matter than other (for recommendations)
- [x] Performer's and tag's photo cropper
- [x] Broken video status for scene (you can add it manually)
- [x] 10-stars rating by half star
- [x] RIP ribbon for death performers 
- [x] Reorder fields of scene edit (for more comfortable filling)
- [x] "Merge from/into other scene..." was added in scene page (it's exists in the scenes list page now, but now it's easier to use in scene page)
- [x] Notes block with saving text to file (.stash/notes.txt) - for saving links to scrape videos in future, for example
- [x] Action for open media file in external player
- [x] In scene edit: highlight already set tags if search of tag contains searchable phrase
- [x] Check file in task before make action (for example, if you will start scan folder and at this moment delete one of video any video in this folder; it has crush before)
- [x] Open tag page instead of filter by tag when click on tag from scene page (because tag page have scenes by this tag already, anf it's more comfortable to edit tag by click on it)
- [x] Filters saves for each scene now and keeps after reload page

### TODO
- [ ] For global recount suggested scenes task add info about status of handled videos
- [ ] Fix scene sprites after convert video

### About Stash
[Original repository](https://github.com/stashapp/stash)

**Stash is a self-hosted webapp written in Go which organizes and serves your porn.**

![demo image](docs/readme_assets/demo_image.png)

* Stash gathers information about videos in your collection from the internet, and is extensible through the use of community-built plugins for a large number of content producers and sites.
* Stash supports a wide variety of both video and image formats.
* You can tag videos and find them later.
* Stash provides statistics about performers, tags, studios and more.

You can [watch a SFW demo video](https://vimeo.com/545323354) to see it in action.

For further information you can consult the [documentation](https://docs.stashapp.cc) or [read the in-app manual](ui/v2.5/src/docs/en).

# How to install reforged version

// TODO: add description how to run server