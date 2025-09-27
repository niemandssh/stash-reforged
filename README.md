# Stash: Reforged

This is changed version of Stash App (from 0.29-dev)!\
I changed original repository for my own personal requests so it could be better than original app (but it harder to run server and maybe a little bit unstable).

_**If you want you can suggest some improvements or functionality - you are welcome**_! (Maybe this functionality will appear faster here than original stash app).

This version will be always in develop branch.

### Some new features

- [x] **Trimmed segments** - allows you to skip start and end of the video with intro or/and advertise (without trim original video)
- [x] **Scenes recommendations** (by weights of tags, performers, studio and groups) - shows you suggested content to watch!
- [x] **Pose tags block** - shows more informative tags to select (for tags with pose flag)
- [x] **"Random" button, "Random best" button** - shows random scene by rating
- [x] **"Review" button** - fore review unorganised scenes without tags or rating
- [x] **Converter broken formats to mp4 and HLS to mp4** - for real converting original video to valid format (like transcode but replaces original video). Convertable video will copy to temp folder (.stash/temp) before file will convert and save correctly (for safe)
- [x] **Tag colors** - mark important tags (color presets and palette of already used colors make color edit more comfortable)
- [x] **O-Count for galleries!** - now you can add count of times for your comics ;) 
- [x] **O-Count stats and graphics**
- [x] **Multiple images for performers** (gallery carousel with set some image as primary) - you can now add 2-3 the best photos!
- [x] **New slideshow and web show modes for galleries** - for comfortable read comics!
- [x] **Primary tag for performer** (will show after name)

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

**For Linux users:**
1. Install `go nodejs bun gcc make ffmpeg`
2. Run `make prepare` for the first time
3. Run `make start` to run server
4. Open http://localhost:9999

\
**For Windows users:**\
Cry)))

_// TODO: add description how to run server for WIN users_ 