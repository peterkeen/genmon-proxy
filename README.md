GenMon Proxy
------------

Proxy GenMon's HTTP API to make it easier to deal with from Home Assistant.

## What?

[GenMon](https://github.com/jgyates/genmon) monitors a backup generator, notifying on significant events like power outages or exercises.
It has built-in support for pushing internal data to MQTT for usage by systems like Home Assistant.

MQTT is not a well supported system in my homeprod infrastructure.
I much prefer Home Assistant to actively poll services rather than rely on a stateful middle tier.

GenMon has a series of HTTP APIs that provide all of the information that the frontend and any other tool might need.
The problem with these APIs is that they're wildly inconsistent, at least in terms of being able to access the data easily from Home Assistant.

Here's an example result from `/cmd/status_json`:

```json
{
  "Status": [
    {
      "Engine": [
        {
          "Switch State": "Auto"
        },
        {
          "Engine State": "Off - Ready"
        },
        {
          "Battery Voltage": "13.00 V"
        },
        {
          "RPM": "0 "
        },
        {
          "Frequency": "0.00 Hz"
        },
        {
          "Output Voltage": "0 V"
        },
        {
          "Active Rotor Poles (Calculated)": "0 "
        }
      ]
    },
    {
      "Line": [
        {
          "Utility Voltage": "242 V"
        },
        {
          "Utility Max Voltage": "246 V"
        },
        {
          "Utility Min Voltage": "240 V"
        },
        {
          "Utility Threshold Voltage": "156 V"
        }
      ]
    },
    {
      "Last Log Entries": {
        "Logs": {
          "Alarm Log": "08/02/23 09:13:55 ESTOP Pressed : Alarm Code: 2800",
          "Service Log": "",
          "Run Log": "10/04/23 10:36:12 Stopped - Auto "
        }
      }
    },
    {
      "Time": [
        {
          "Monitor Time": "Tuesday October 10, 2023 16:05:38"
        },
        {
          "Generator Time": "Tuesday October 10, 2023 16:05"
        }
      ]
    }
  ]
}
```

There are a couple of weird things about that. First, _most_ of the data is presented as hashes with one key, sometimes arranged as an array.
But then there's those logs that are a normal hash.

Here's another example, this time `/cmd/outage_json`:

```json
{
  "Outage": [    {
      "Status": "No outage has occurred since program launched."
    },
    {
      "System In Outage": "No"
    },
    {
      "Utility Voltage": "244 V"
    },
    {
      "Utility Voltage Minimum": "240 V"
    },
    {
      "Utility Voltage Maximum": "246 V"
    },
    {
      "Utility Threshold Voltage": "156 V"
    },
    {
      "Utility Pickup Voltage": "190 V"
    },
    {
      "Startup Delay": "5 s"
    },
    {
      "Outage Log": [
        "09-01-2023 10:35:42, Duration: 0:00:27",
        "08-24-2023 19:50:05, Duration: 4:07:28",
        "08-24-2023 18:48:08, Duration: 0:58:46",
        "08-24-2023 18:34:10, Duration: 0:00:36"
      ]
    }
  ]
}
```

So, "normal" array of single pair hashes. But then we get to `Outage Log` and it's an array of strings! Cool. Cool cool cool.

## Ok, what's better?

Here's the output from `genmon-proxy`:

```json
{
  "maintenance_ambient_temperature_sensor": "44 F",
  "maintenance_controller_detected": "Evolution 2.0, Air Cooled",
  "maintenance_controller_settings_calibrate_current_1": "1504 ",
  "maintenance_controller_settings_calibrate_current_2": "1504 ",
  "maintenance_controller_settings_calibrate_volts": "1047 ",
  "maintenance_controller_settings_nominal_line_voltage": "Unknown V",
  "maintenance_controller_settings_rated_max_power": "Unknown kW",
  "maintenance_exercise_exercise_time": "Weekly Wednesday 10:30 Quiet Mode Off",
  "maintenance_fuel_type": "Natural Gas",
  "maintenance_generator_phase": "Unknown",
  "maintenance_generator_serial_number": "None - Controller has been replaced",
  "maintenance_model": "Unknown",
  "maintenance_nominal_frequency": "60 Hz",
  "maintenance_nominal_rpm": "3600",
  "maintenance_rated_k_w": "Unknown kW",
  "maintenance_service_battery_check_due": "04/18/2024 ",
  "maintenance_service_firmware_version": "V1.15",
  "maintenance_service_hardware_version": "V1.03",
  "maintenance_service_service_a_due": "111 hrs or 04/18/2025 ",
  "maintenance_service_service_b_due": "311 hrs or 04/18/2027 ",
  "maintenance_service_total_run_hours": "88.00 h",
  "monitor_communication_stats_average_transaction_time": "0.0340 sec",
  "monitor_communication_stats_comm_restarts": "2",
  "monitor_communication_stats_crc_errors": "0 ",
  "monitor_communication_stats_crc_percent_errors": "0.00%",
  "monitor_communication_stats_discarded_bytes": "0",
  "monitor_communication_stats_invalid_data": "0",
  "monitor_communication_stats_modbus_exceptions": "391",
  "monitor_communication_stats_modbus_transport": "TCP",
  "monitor_communication_stats_packet_count": "M: 5597712, S: 5597704",
  "monitor_communication_stats_packets_per_second": "58.31",
  "monitor_communication_stats_sync_errors": "0",
  "monitor_communication_stats_timeout_errors": "7",
  "monitor_communication_stats_timeout_percent_errors": "0.00%",
  "monitor_communication_stats_validation_errors": "0",
  "monitor_generator_monitor_stats_controller": "Evolution 2.0, Air Cooled",
  "monitor_generator_monitor_stats_generator_monitor_version": "V1.18.18",
  "monitor_generator_monitor_stats_monitor_health": "OK",
  "monitor_generator_monitor_stats_run_time": "Generator Monitor running for 2 days, 5:20:08.",
  "monitor_generator_monitor_stats_update_available": "No",
  "monitor_platform_stats_cpu_utilization": "20.87%",
  "monitor_platform_stats_network_interface_used": "",
  "monitor_platform_stats_os_name": "Ubuntu",
  "monitor_platform_stats_os_version": "22.04.3 LTS (Jammy Jellyfish)",
  "monitor_platform_stats_system_time": "Tuesday October 10, 2023 21:21:14",
  "monitor_platform_stats_system_uptime": "99 days, 7:04:13",
  "outage_outage_log": "08-24-2023 18:34:10, Duration: 0:00:36",
  "outage_startup_delay": "5 s",
  "outage_status": "No outage has occurred since program launched.",
  "outage_system_in_outage": "No",
  "outage_utility_pickup_voltage": "190 V",
  "outage_utility_threshold_voltage": "156 V",
  "outage_utility_voltage": "243 V",
  "outage_utility_voltage_maximum": "246 V",
  "outage_utility_voltage_minimum": "239 V",
  "status_engine_active_rotor_poles_calculated_type": "int",
  "status_engine_active_rotor_poles_calculated_unit": "",
  "status_engine_active_rotor_poles_calculated_value": "0",
  "status_engine_battery_voltage_type": "float",
  "status_engine_battery_voltage_unit": "V",
  "status_engine_battery_voltage_value": "13",
  "status_engine_engine_state": "Off - Ready",
  "status_engine_frequency_type": "float",
  "status_engine_frequency_unit": "Hz",
  "status_engine_frequency_value": "0",
  "status_engine_output_voltage_type": "int",
  "status_engine_output_voltage_unit": "V",
  "status_engine_output_voltage_value": "0",
  "status_engine_rpm_type": "int",
  "status_engine_rpm_unit": "",
  "status_engine_rpm_value": "0",
  "status_engine_switch_state": "Auto",
  "status_last_log_entries_logs_alarm_log": "08/02/23 09:13:55 ESTOP Pressed : Alarm Code: 2800",
  "status_last_log_entries_logs_run_log": "10/04/23 10:36:12 Stopped - Auto ",
  "status_last_log_entries_logs_service_log": "",
  "status_line_utility_max_voltage_type": "int",
  "status_line_utility_max_voltage_unit": "V",
  "status_line_utility_max_voltage_value": "246",
  "status_line_utility_min_voltage_type": "int",
  "status_line_utility_min_voltage_unit": "V",
  "status_line_utility_min_voltage_value": "239",
  "status_line_utility_threshold_voltage_type": "int",
  "status_line_utility_threshold_voltage_unit": "V",
  "status_line_utility_threshold_voltage_value": "156",
  "status_line_utility_voltage_type": "int",
  "status_line_utility_voltage_unit": "V",
  "status_line_utility_voltage_value": "243",
  "status_time_generator_time": "Tuesday October 10, 2023 21:21",
  "status_time_monitor_time": "Tuesday October 10, 2023 21:21:14"
}
```

Ah, that's better. A single flat hash of string keys and values containing the results from four different upstream APIs.
This is _very_ easy to deal with in Home Assistant:

```yaml
rest:
  - resource: http://genmon-proxy/
    scan_interval: 10
    sensor:
      - name: "Generator Engine Status"
        value_template: "{{ value_json.status_engine_engine_state }}"
    binary_sensor:
      - name: "Generator Outage Status"
        value_template: "{{ value_json.outage_system_in_outate }}"
        
    # ... and so on
```

## Tailscale!?

Sure why not?
This thing requires a `TS_AUTHTOKEN` environment variable to be set and it can only talk to an upstream GenMon that has been exposed on the tailnet. 

If you want to fork it and strip that stuff out, be my guest!

## License

MIT
