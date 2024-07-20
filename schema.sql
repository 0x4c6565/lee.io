CREATE TABLE `mac_oui` (
  `id` int NOT NULL AUTO_INCREMENT,
  `oui` text NOT NULL,
  `company_name` text NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `bgp_route` (
  `id` varchar(36) NOT NULL PRIMARY KEY,
  `version` int(11) NOT NULL,
  `ip_version` int(1) NOT NULL,
  `route` text NOT NULL,
  `asn_number` bigint(20) NOT NULL,
  `owner` text NOT NULL,
  `country_code` varchar(10) NOT NULL,
  `ipv4_start` bigint(20) NOT NULL,
  `ipv4_end` bigint(20) NOT NULL,
  `ipv6_start` varchar(45) NOT NULL,
  `ipv6_end` varchar(45) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `bgp_route_version` (
  `version` int(11) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;