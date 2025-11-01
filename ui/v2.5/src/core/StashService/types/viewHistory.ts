import { gql } from "graphql-tag";

export const FIND_VIEW_HISTORY = gql`
  query FindViewHistory($filter: FindFilterType, $historyFilter: ViewHistoryFilter) {
    findViewHistory(filter: $filter, history_filter: $historyFilter) {
      count
      items {
        scene {
          id
          title
          play_count
          resume_time
          force_hls
          is_broken
          is_probably_broken
          is_not_broken
               files {
                 path
                 duration
               }
          paths {
            screenshot
            preview
            vtt
          }
          performers {
            id
            name
            gender
          }
          studio {
            id
            name
            image_path
          }
        }
        gallery {
          id
          title
          code
          date
          details
          photographer
          rating100
          organized
          pinned
          o_counter
          play_count
          view_history
          display_mode
          image_count
          paths {
            cover
            preview
          }
          performers {
            id
            name
            gender
          }
          studio {
            id
            name
            image_path
          }
          files {
            id
            path
            size
            mod_time
            fingerprints {
              type
              value
            }
          }
        }
        viewDate
        oDate
        viewCount
      }
    }
  }
`;