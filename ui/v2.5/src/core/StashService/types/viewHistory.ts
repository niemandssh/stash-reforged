import gql from "graphql-tag";

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
        viewDate
        oDate
        viewCount
      }
    }
  }
`;