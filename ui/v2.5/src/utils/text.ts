import { IntlShape } from "react-intl";

// Typescript currently does not implement the intl Unit interface
type Unit =
  | "byte"
  | "kibibyte"
  | "mebibyte"
  | "gibibyte"
  | "tebibyte"
  | "pebibyte";
const Units: Unit[] = [
  "byte",
  "kibibyte",
  "mebibyte",
  "gibibyte",
  "tebibyte",
  "pebibyte",
];
const shortUnits = ["B", "KiB", "MiB", "GiB", "TiB", "PiB"];

const fileSize = (bytes: number = 0) => {
  if (Number.isNaN(parseFloat(String(bytes))) || !Number.isFinite(bytes))
    return { size: 0, unit: Units[0] };

  let unit = 0;
  let count = bytes;
  // calculating base 2 units
  while (count >= 1024 && unit + 1 < Units.length) {
    count /= 1024;
    unit++;
  }

  return {
    size: count,
    unit: Units[unit],
  };
};

class DurationUnit {
  static readonly SECOND: DurationUnit = new DurationUnit(
    "second",
    "seconds",
    "s",
    1
  );
  static readonly MINUTE: DurationUnit = new DurationUnit(
    "minute",
    "minutes",
    "m",
    60
  );
  static readonly HOUR: DurationUnit = new DurationUnit(
    "hour",
    "hours",
    "h",
    DurationUnit.MINUTE.secs * 60
  );
  static readonly DAY: DurationUnit = new DurationUnit(
    "day",
    "days",
    "D",
    DurationUnit.HOUR.secs * 24
  );
  static readonly WEEK: DurationUnit = new DurationUnit(
    "week",
    "weeks",
    "W",
    DurationUnit.DAY.secs * 7
  );
  static readonly MONTH: DurationUnit = new DurationUnit(
    "month",
    "months",
    "M",
    DurationUnit.DAY.secs * 30
  );
  static readonly YEAR: DurationUnit = new DurationUnit(
    "year",
    "years",
    "Y",
    DurationUnit.DAY.secs * 365
  );

  static readonly DURATIONS: DurationUnit[] = [
    DurationUnit.SECOND,
    DurationUnit.MINUTE,
    DurationUnit.HOUR,
    DurationUnit.DAY,
    DurationUnit.WEEK,
    DurationUnit.MONTH,
    DurationUnit.YEAR,
  ];

  private constructor(
    private readonly singular: string,
    private readonly plural: string,
    private readonly shortString: string,
    public secs: number
  ) {}

  toString() {
    return this.shortString;
  }
}

class DurationCount {
  public constructor(
    public readonly count: number,
    public readonly duration: DurationUnit
  ) {}

  toString() {
    return this.count.toString() + this.duration.toString();
  }
}

const secondsAsTime = (seconds: number = 0): DurationCount[] => {
  if (Number.isNaN(parseFloat(String(seconds))) || !Number.isFinite(seconds))
    return [new DurationCount(0, DurationUnit.DURATIONS[0])];

  const result = [];
  let remainingSeconds = seconds;
  // Run down the possible durations and pull them out
  for (let i = DurationUnit.DURATIONS.length - 1; i >= 0; i--) {
    const q = Math.floor(remainingSeconds / DurationUnit.DURATIONS[i].secs);
    if (q !== 0) {
      remainingSeconds %= DurationUnit.DURATIONS[i].secs;
      result.push(new DurationCount(q, DurationUnit.DURATIONS[i]));
    }
  }
  return result;
};

const secondsAsTimeString = (
  seconds: number = 0,
  maxUnitCount: number = 2
): string => {
  return secondsAsTime(seconds).slice(0, maxUnitCount).join(" ");
};

const formatFileSizeUnit = (u: Unit) => {
  const i = Units.indexOf(u);
  return shortUnits[i];
};

// returns the number of fractional digits to use when displaying file sizes
// returns 0 for MB and under, 1 for GB and over.
const fileSizeFractionalDigits = (unit: Unit) => {
  if (Units.indexOf(unit) >= 3) {
    return 1;
  }

  return 0;
};

// Converts seconds to a [hh:]mm:ss[.ffff] where hh is only shown if hours is non-zero,
// and ffff is shown only if frameRate is set, and the seconds includes a fractional component.
// A negative input will result in a -hh:mm:ss or -mm:ss output.
const secondsToTimestamp = (secondsInput: number, includeMS?: boolean) => {
  let neg = false;
  if (secondsInput < 0) {
    neg = true;
    secondsInput = -secondsInput;
  }

  const fracSeconds = secondsInput % 1;
  const ms = Math.round(fracSeconds * 1000);

  let seconds = Math.trunc(secondsInput);

  const s = seconds % 60;
  seconds = (seconds - s) / 60;

  const m = seconds % 60;
  seconds = (seconds - m) / 60;

  const h = seconds;

  let ret = String(s).padStart(2, "0");
  if (h === 0) {
    ret = String(m) + ":" + ret;
  } else {
    ret = String(m).padStart(2, "0") + ":" + ret;
    ret = String(h) + ":" + ret;
  }

  if (includeMS && ms > 0) {
    ret += "." + ms.toString().padStart(3, "0");
  }

  if (neg) {
    return "-" + ret;
  } else {
    return ret;
  }
};

const formatTimestampRange = (start: number, end: number | undefined) => {
  if (end === undefined) {
    return secondsToTimestamp(start);
  }
  return `${secondsToTimestamp(start)}-${secondsToTimestamp(end)}`;
};

const timestampToSeconds = (v: string | null | undefined) => {
  if (!v) {
    return null;
  }

  const splits = v.split(":");

  if (splits.length > 3) {
    return null;
  }

  let secondsPart = splits[splits.length - 1];
  let msFrac = 0;
  if (secondsPart.includes(".")) {
    const secondsParts = secondsPart.split(".");
    if (secondsParts.length !== 2) {
      return null;
    }

    secondsPart = secondsParts[0];

    const msPart = parseInt(secondsParts[1], 10);
    if (Number.isNaN(msPart)) {
      return null;
    }

    msFrac = msPart / 1000;
  }

  let seconds = 0;
  let factor = 1;
  while (splits.length > 0) {
    const thisSplit = splits.pop();
    if (thisSplit === undefined) {
      return null;
    }

    const thisInt = parseInt(thisSplit, 10);
    if (Number.isNaN(thisInt)) {
      return null;
    }

    seconds += factor * thisInt;
    factor *= 60;
  }

  return seconds + msFrac;
};

const fileNameFromPath = (path: string) => {
  if (!!path === false) return "No File Name";
  return path.replace(/^.*[\\/]/, "");
};

/** Parses a date string in multiple formats. Returns { year, month (1-12), day (1-31) } or null. */
const parseFlexibleDateString = (
  dateString: string
): { year: number; month: number; day: number } | null => {
  const s = dateString.trim();
  if (!s) return null;

  // YYYY (only year — use mid-year 1 June for calculations)
  if (/^\d{4}$/.test(s)) {
    const year = Number(s);
    if (year >= 1900 && year <= 2100) return { year, month: 6, day: 1 };
    return null;
  }

  // YYYY-MM or YYYY-MM-DD (with -)
  const dashParts = s.split("-");
  if (dashParts.length >= 2 && dashParts.length <= 3) {
    const a = Number(dashParts[0]);
    const b = Number(dashParts[1]);
    const c = dashParts[2] !== undefined ? Number(dashParts[2]) : 1;
    // YYYY-MM-DD or YYYY-MM
    if (dashParts[0].length === 4 && a >= 1900 && a <= 2100) {
      const month = dashParts.length >= 2 ? Math.max(1, Math.min(12, b)) : 1;
      const day = dashParts.length === 3 ? Math.max(1, Math.min(31, c)) : 1;
      return { year: a, month, day };
    }
    // DD-MM-YYYY
    if (
      dashParts[2] !== undefined &&
      dashParts[2].length === 4 &&
      a >= 1 && a <= 31 &&
      b >= 1 && b <= 12
    ) {
      const year = Number(dashParts[2]);
      if (year >= 1900 && year <= 2100) return { year, month: b, day: a };
    }
  }

  // DD.MM.YYYY or DD/MM/YYYY
  const dotParts = s.split(".");
  const slashParts = s.split("/");
  const parts = dotParts.length === 3 ? dotParts : slashParts.length === 3 ? slashParts : null;
  if (parts && parts[2].length === 4) {
    const day = Number(parts[0]);
    const month = Number(parts[1]);
    const year = Number(parts[2]);
    if (
      day >= 1 && day <= 31 &&
      month >= 1 && month <= 12 &&
      year >= 1900 && year <= 2100
    ) {
      return { year, month, day };
    }
  }

  // MM.YYYY or M.YYYY (month and year)
  if (dotParts.length === 2 && dotParts[1].length === 4) {
    const month = Number(dotParts[0]);
    const year = Number(dotParts[1]);
    if (month >= 1 && month <= 12 && year >= 1900 && year <= 2100) {
      return { year, month, day: 1 };
    }
  }

  // MM-YYYY or M-YYYY (month and year; 2 hyphen parts, second is 4 digits)
  if (dashParts.length === 2 && dashParts[1].length === 4) {
    const month = Number(dashParts[0]);
    const year = Number(dashParts[1]);
    if (month >= 1 && month <= 12 && year >= 1900 && year <= 2100) {
      return { year, month, day: 1 };
    }
  }

  return null;
};

/** Returns true if the string is empty or a valid date in any supported format (optional + multiple formats). */
const isValidDateString = (value: string | null | undefined): boolean => {
  if (value == null || value.trim() === "") return true;
  return parseFlexibleDateString(value.trim()) !== null;
};

const stringToDate = (dateString: string) => {
  if (!dateString) return null;

  const parsed = parseFlexibleDateString(dateString);
  if (!parsed) return null;
  if (parsed.month === 0 || parsed.day === 0) return null;

  return new Date(
    parsed.year,
    parsed.month - 1,
    parsed.day,
    0,
    0,
    0,
    0
  );
};

const stringToFuzzyDate = (dateString: string) => {
  if (!dateString) return null;

  const parsed = parseFlexibleDateString(dateString);
  if (!parsed) {
    // Fallback: try legacy hyphen-only parsing for partial YYYY / YYYY-MM
    const parts = dateString.trim().split("-");
    let year = Number(parts[0]);
    if (isNaN(year) || year < 1900 || year > 2100)
      year = new Date().getFullYear();
    // Only year -> mid-year (1 June); YYYY-MM -> 1st of month
    let month = parts.length > 1 ? Math.max(1, Math.min(12, Number(parts[1]) || 1)) : 6;
    const day = parts.length > 2 ? Math.max(1, Math.min(31, Number(parts[2]) || 1)) : 1;
    return new Date(year, month - 1, day, 0, 0, 0, 0);
  }

  return new Date(
    parsed.year,
    parsed.month - 1,
    parsed.day,
    0,
    0,
    0,
    0
  );
};

const stringToFuzzyDateTime = (dateString: string) => {
  if (!dateString) return null;

  const dateTime = dateString.split(" ");

  let date: Date | null = null;
  if (dateTime.length > 0) {
    date = stringToFuzzyDate(dateTime[0]);
  }

  if (!date) {
    date = new Date();
  }

  if (dateTime.length > 1) {
    const timeParts = dateTime[1].split(":");
    if (date && timeParts.length > 0) {
      date.setHours(Number(timeParts[0]));
    }
    if (date && timeParts.length > 1) {
      date.setMinutes(Number(timeParts[1]));
    }
    if (date && timeParts.length > 2) {
      date.setSeconds(Number(timeParts[2]));
    }
  }

  return date;
};

function dateToString(date: Date) {
  return `${date.getFullYear()}-${(date.getMonth() + 1)
    .toString()
    .padStart(2, "0")}-${date.getDate().toString().padStart(2, "0")}`;
}

function dateTimeToString(date: Date) {
  return `${dateToString(date)} ${date
    .getHours()
    .toString()
    .padStart(2, "0")}:${date.getMinutes().toString().padStart(2, "0")}`;
}

const getAge = (dateString?: string | null, fromDateString?: string | null) => {
  if (!dateString) return 0;

  const birthdate = stringToDate(dateString);
  const fromDate = fromDateString ? stringToDate(fromDateString) : new Date();

  if (!birthdate || !fromDate) return 0;

  let age = fromDate.getFullYear() - birthdate.getFullYear();
  if (
    birthdate.getMonth() > fromDate.getMonth() ||
    (birthdate.getMonth() >= fromDate.getMonth() &&
      birthdate.getDate() > fromDate.getDate())
  ) {
    age -= 1;
  }

  return age;
};

const bitRate = (bitrate: number) => {
  const megabits = bitrate / 1000000;
  return `${megabits.toFixed(2)} megabits per second`;
};

const resolution = (width: number, height: number) => {
  const number = width > height ? height : width;
  if (number >= 6144) {
    return "HUGE";
  }
  if (number >= 3840) {
    return "8K";
  }
  if (number >= 3584) {
    return "7K";
  }
  if (number >= 3000) {
    return "6K";
  }
  if (number >= 2560) {
    return "5K";
  }
  if (number >= 1920) {
    return "4K";
  }
  if (number >= 1440) {
    return "1440p";
  }
  if (number >= 1080) {
    return "1080p";
  }
  if (number >= 720) {
    return "720p";
  }
  if (number >= 540) {
    return "540p";
  }
  if (number >= 480) {
    return "480p";
  }
  if (number >= 360) {
    return "360p";
  }
  if (number >= 240) {
    return "240p";
  }
  if (number >= 144) {
    return "144p";
  }
};

const sanitiseURL = (url?: string, siteURL?: URL) => {
  if (!url) {
    return url;
  }

  if (url.startsWith("http://") || url.startsWith("https://")) {
    // just return the entire URL
    return url;
  }

  if (siteURL) {
    // if url starts with the site host, then prepend the protocol
    if (url.startsWith(siteURL.host)) {
      return `${siteURL.protocol}//${url}`;
    }

    // otherwise, construct the url from the protocol, host and passed url
    return `${siteURL.protocol}//${siteURL.host}/${url}`;
  }

  // just prepend the protocol - assume https
  return `https://${url}`;
};

const domainFromURL = (urlString?: string, url?: URL) => {
  if (url) {
    return url.hostname;
  } else if (urlString) {
    var urlDomain = "";
    try {
      var sanitizedUrl = sanitiseURL(urlString);
      if (sanitizedUrl) {
        urlString = sanitizedUrl;
      }
      urlDomain = new URL(urlString).hostname;
    } catch {
      urlDomain = urlString; // We cant determine the hostname so we return the base string
    }
    return urlDomain;
  } else {
    return "";
  }
};

/**
 * Returns display string for a date when API provides explicit date_display
 * (e.g. "YYYY" or "YYYY-MM" for partial dates). When dateDisplay is set, use it;
 * otherwise format the full date string.
 */
const getDateDisplayString = (
  dateStr: string | null | undefined,
  dateDisplay?: string | null
): string => {
  if (dateDisplay != null && dateDisplay !== "") {
    return dateDisplay;
  }
  if (!dateStr) return "";
  return dateStr.trim();
};

/**
 * Returns the string to use in date edit inputs. When date_display is set (partial date),
 * use it so the form shows "2001-06" or "2001" instead of "2001-06-01"; otherwise use the full date.
 * This avoids re-submitting a full date when the user only had a partial date and didn't change it.
 */
const getDateEditString = (
  dateStr: string | null | undefined,
  dateDisplay?: string | null
): string => {
  if (dateDisplay != null && dateDisplay !== "") {
    return dateDisplay;
  }
  if (!dateStr) return "";
  return dateStr.trim();
};

/**
 * Formats a date for display. When dateDisplay is provided (from API), shows
 * only year or month+year; otherwise formats the full date.
 */
const formatDate = (
  intl: IntlShape,
  date?: string,
  utc = true,
  dateDisplay?: string | null
) => {
  if (!date && !dateDisplay) return "";
  if (dateDisplay != null && dateDisplay !== "") {
    if (dateDisplay.length === 4) return dateDisplay;
    if (dateDisplay.length === 7) {
      const year = Number(dateDisplay.slice(0, 4));
      const monthIndex = Number(dateDisplay.slice(5, 7)) - 1;
      const d = utc
        ? new Date(Date.UTC(year, monthIndex, 1))
        : new Date(year, monthIndex, 1);
      return intl.formatDate(d, {
        year: "numeric",
        month: "long",
        timeZone: utc ? "utc" : undefined,
      });
    }
  }
  if (!date) return "";
  return intl.formatDate(date, {
    format: "long",
    timeZone: utc ? "utc" : undefined,
  });
};

const formatDateTime = (intl: IntlShape, dateTime?: string, utc = false) =>
  `${formatDate(intl, dateTime, utc)} ${intl.formatTime(dateTime, {
    timeZone: utc ? "utc" : undefined,
  })}`;

type CountUnit = "" | "K" | "M" | "B";
const CountUnits: CountUnit[] = ["", "K", "M", "B"];

const abbreviateCounter = (counter: number = 0) => {
  if (Number.isNaN(parseFloat(String(counter))) || !Number.isFinite(counter))
    return { size: 0, unit: CountUnits[0] };

  let unit = 0;
  let digits = 0;
  let count = counter;
  while (count >= 1000 && unit + 1 < CountUnits.length) {
    count /= 1000;
    unit++;
    digits = 1;
  }

  return {
    size: count,
    unit: CountUnits[unit],
    digits: digits,
  };
};

/*
 * Trims quotes if the text has leading/trailing quotes
 */
const stripQuotes = (text: string) => {
  if (text.startsWith('"') && text.endsWith('"')) return text.slice(1, -1);
  return text;
};

/*
 * Wraps string in quotes
 */
const addQuotes = (text: string) => `"${text}"`;

const TextUtils = {
  fileSize,
  formatFileSizeUnit,
  fileSizeFractionalDigits,
  secondsToTimestamp,
  formatTimestampRange,
  timestampToSeconds,
  fileNameFromPath,
  stringToDate,
  stringToFuzzyDate,
  stringToFuzzyDateTime,
  dateToString,
  dateTimeToString,
  isValidDateString,
  getDateDisplayString,
  getDateEditString,
  age: getAge,
  bitRate,
  resolution,
  sanitiseURL,
  domainFromURL,
  formatDate,
  formatDateTime,
  secondsAsTimeString,
  abbreviateCounter,
  stripQuotes,
  addQuotes,
};

export default TextUtils;
