import React, { useMemo } from "react";
import { useOCountStats, useStats } from "src/core/StashService";
import { LoadingIndicator } from "src/components/Shared/LoadingIndicator";
import Chart from "react-apexcharts";
import { ApexOptions } from "apexcharts";
import { FormattedMessage, FormattedNumber, useIntl } from "react-intl";
import "./OCountStats.scss";

interface IDataPoint {
  date: string;
  date_display: string;
  count: number;
}

const OCountChart: React.FC<{ data: IDataPoint[] }> = ({ data }) => {
  const intl = useIntl();

  const xaxisCategories = useMemo(() => {
    if (!data || data.length === 0) return [];
    return data.map((point) => point.date_display);
  }, [data]);

  const chartOptions = useMemo((): ApexOptions => {
    return {
      chart: {
        type: "area",
        height: 350,
        toolbar: {
          show: true,
          tools: {
            zoom: true,
            zoomin: true,
            zoomout: true,
            pan: true,
            reset: true,
          },
        },
        zoom: {
          enabled: true,
          type: "x",
          autoScaleYaxis: true,
        },
        events: {
          mounted: function (chartContext) {
            if (data && data.length > 0) {
              // Зум на последние 30 дней от последней точки в данных
              const lastIndex = data.length - 1;
              const lastMonthStart = Math.max(0, lastIndex - 29);
              chartContext.zoomX(lastMonthStart, lastIndex + 1);
            }
          },
        },
        animations: {
          enabled: true,
          speed: 800,
        },
        parentHeightOffset: 0,
        redrawOnParentResize: true,
      },
      dataLabels: {
        enabled: false,
      },
      stroke: {
        curve: "smooth",
        width: 2,
        colors: ["#007bff"],
      },
      fill: {
        type: "gradient",
        gradient: {
          shadeIntensity: 1,
          opacityFrom: 0.3,
          opacityTo: 0.1,
          stops: [0, 100],
          colorStops: [
            {
              offset: 0,
              color: "#007bff",
              opacity: 0.3,
            },
            {
              offset: 100,
              color: "#007bff",
              opacity: 0.1,
            },
          ],
        },
      },
      xaxis: {
        type: "category",
        categories: xaxisCategories,
        tickAmount: data ? data.length : 0,
        tickPlacement: "on",
        labels: {
          style: {
            colors: "#666",
            fontSize: "12px",
          },
          rotate: -45,
          rotateAlways: true,
          hideOverlappingLabels: false,
          show: true,
          trim: false,
        },
        axisBorder: {
          show: false,
        },
        axisTicks: {
          show: false,
        },
        tooltip: {
          enabled: true,
          style: {
            fontSize: "12px",
            fontFamily: "inherit",
          },
        },
      },
      yaxis: {
        min: 0,
        labels: {
          style: {
            colors: "#666",
            fontSize: "12px",
          },
        },
        title: {
          text: intl.formatMessage({ id: "o_count_stats.y_axis_title" }),
          style: {
            color: "#666",
            fontSize: "14px",
          },
        },
      },
      grid: {
        borderColor: "#e0e0e0",
        strokeDashArray: 4,
        xaxis: {
          lines: {
            show: false,
          },
        },
        yaxis: {
          lines: {
            show: true,
          },
        },
      },
      tooltip: {
        enabled: true,
        shared: false,
        intersect: false,
        followCursor: true,
        fixed: {
          enabled: false,
        },
        custom: function ({ series, seriesIndex, dataPointIndex }) {
          const value = series[seriesIndex][dataPointIndex];
          const dateDisplay = data[dataPointIndex]
            ? data[dataPointIndex].date_display
            : "";

          const timesText = intl.formatMessage(
            { id: "o_count_stats.tooltip_times" },
            { count: value }
          );

          return `
              <div class="apex-tooltip">
                <div class="tooltip-date">
                  ${dateDisplay}
                </div>
                <div class="tooltip-count">
                  ${value} ${timesText}
                </div>
              </div>
            `;
        },
      },
      markers: {
        size: 4,
        colors: ["#007bff"],
        strokeColors: "#fff",
        strokeWidth: 2,
        hover: {
          size: 6,
        },
      },
      colors: ["#007bff"],
    };
  }, [intl, xaxisCategories, data]);

  const series = useMemo(() => {
    if (!data || data.length === 0) return [];

    return [
      {
        name: "O-Count",
        data: data.map((point) => ({
          x: point.date,
          y: point.count,
        })),
      },
    ];
  }, [data]);

  if (!data || data.length === 0) {
    return (
      <div className="text-center mt-5">
        <p>
          <FormattedMessage id="o_count_stats.no_data" />
        </p>
      </div>
    );
  }

  return (
    <div className="mt-4">
      <div className="chart-container">
        <Chart
          options={chartOptions}
          series={series}
          type="area"
          height={350}
        />
      </div>

      <div className="text-center mt-3">
        <small className="text-muted">
          <FormattedMessage id="o_count_stats.description" />
        </small>
      </div>
    </div>
  );
};

export const OCountStats: React.FC = () => {
  const { data, error, loading } = useOCountStats();
  const { data: generalData } = useStats();

  // Вычисляем O-Count за текущий месяц
  const currentMonthOCount = useMemo(() => {
    if (!data?.oCountStats?.daily_stats) return 0;
    const now = new Date();
    const currentYear = now.getFullYear();
    const currentMonth = now.getMonth() + 1;

    return data.oCountStats.daily_stats.reduce((sum, point) => {
      const pointDate = new Date(point.date);
      if (
        pointDate.getFullYear() === currentYear &&
        pointDate.getMonth() + 1 === currentMonth
      ) {
        return sum + point.count;
      }
      return sum;
    }, 0);
  }, [data]);

  // Вычисляем O-Count за последние 30 дней
  const last30DaysOCount = useMemo(() => {
    if (!data?.oCountStats?.daily_stats) return 0;
    const now = new Date();
    const thirtyDaysAgo = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000);

    return data.oCountStats.daily_stats.reduce((sum, point) => {
      const pointDate = new Date(point.date);
      if (pointDate >= thirtyDaysAgo) {
        return sum + point.count;
      }
      return sum;
    }, 0);
  }, [data]);

  if (error) return <span>{error.message}</span>;
  if (loading || !data) return <LoadingIndicator />;

  return (
    <div className="mt-5">
      <div className="col col-sm-8 m-sm-auto row stats mb-4">
        <div className="stats-element">
          <p className="title">
            <FormattedNumber value={generalData?.stats.total_o_count || 0} />
          </p>
          <p className="heading">
            <FormattedMessage id="stats.total_o_count" />
          </p>
        </div>
        <div className="stats-element">
          <p className="title">
            <FormattedNumber value={currentMonthOCount} />
          </p>
          <p className="heading">
            <FormattedMessage id="o_count_stats.current_month" />
          </p>
        </div>
        <div className="stats-element">
          <p className="title">
            <FormattedNumber value={last30DaysOCount} />
          </p>
          <p className="heading">
            <FormattedMessage id="o_count_stats.last_30_days" />
          </p>
        </div>
      </div>

      <div className="row mt-3">
        <div className="col-12">
          <hr className="mb-4" />
          <h4 className="text-center mb-4 mt-5">
            <FormattedMessage id="o_count_stats.title" />
          </h4>
          <OCountChart data={data.oCountStats.daily_stats} />
        </div>
      </div>
    </div>
  );
};
