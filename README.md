# Stash: Reforged

This is changed version of Stash App (from 0.29-dev)!\
I changed original repository for my own personal requests so it could be better than original app (but it harder to run server and maybe a little bit unstable).

_**If you want you can suggest some improvements or functionality - you are welcome**_! (Maybe this functionality will appear faster here than original stash app).

This version will be always in develop branch.

### Some new features

- [x] **Scenes recommendations** (by weights of tags, performers, studio and groups) - shows you suggested content to watch!
- [x] **New slideshow and web show modes for galleries** - for comfortable read comics!
- [x] **Galleries views** - count views like in scenes! View counts automatically only for "Web mode" when you scroll after 3rd image (once per gallery open)
- [x] **O-Count for galleries!** - now you can add count of times for your comics ;)
- [x] **Trimmed segments** - allows you to skip start and end of the video with intro or/and advertise (without trim original video)
- [x] **Scene views history page** - you can view list of scene views like in YouTube! Grouping of views in a row and o-count marker shows in a history card 
- [x] **Play markers mode in scene** - allows you to play only segments by one marker (for example, if a marker has 3 segments, they will be played one after another with automatic jumps to the next segment) sequentially or play all marker segments at once. This mode is useful for large movies with overlapping scenes where you want to watch only one scene, or for viewing only interesting content
- [x] **HLS correction** - special checkbox in Scene filters tab that correct sound in HLS MP4 direct stream (these videos is broken but this mode can fix it, buffering delay only on start of play. Original transcodes was very laggy for me)
- [x] **Pose tags block** - shows more informative tags to select (for tags with pose flag)
- [x] **"Random" button, "Random best" button** - shows random scene by rating
- [x] **"Review" button** - fore review unorganised scenes without tags or rating
- [x] **Converter broken formats to mp4 and HLS to mp4** - for real converting original video to valid format (like transcode but replaces original video). Convertable video will copy to temp folder (.stash/temp) before file will convert and save correctly (for safe)
- [x] **Convert - Trim video** - re-encode video with trimming based on trimmed segments (start time and end time)
- [x] **Convert - Reduce resolution** - you can reduce resolution for big videos (for example, 4k to 2k or fullhd) if you don't want to store big files
- [x] **Tag colors** - mark important tags (color presets and palette of already used colors make color edit more comfortable) 
- [x] **O-Count stats and graphics**
- [x] **Multiple images for performers** (gallery carousel with set some image as primary) - you can now add 2-3 the best photos!
- [x] **Primary tag for performer** (will show after name)
- [x] **Small roles** - show performer in small badge in "Also starred" block after Performers in scene (for example, for hiding mens from main performers list, you anyway don't even try to fill men's photos or something else)
- [x] **Pins for scenes and galleries** - you can pin important scenes or galleries in the list
- [x] **Next scene with timer** - after the end of video like in YouTube (selects one of top-5 suggested scenes). You can change time or turn it off in Settings - Custom
- [x] **New selective scan** - now you can select not only folders, but also individual files for scanning. New convenient Grid mode, selected files and folders are now displayed on the right side 
- [x] **Separate tags for performers in scene** - useful for add tags by each performer if there is 3 or more performers in scene and you need to describe each of them

**And a lot of other improvements! You can view the full list [in this page](IMPROVEMENTS.md).**

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

1. **Use compiled version from releases (for all platforms)**

2. **For Linux users:**
    1. Install `go nodejs bun gcc make ffmpeg`
    2. Run `make prepare` for the first time
    3. Run `make start` to run server
    4. Open http://localhost:9999
