-- +goose Up
-- SQL in this section is executed when the migration is applied.
-- phpMyAdmin SQL Dump
-- version 4.6.5.2
-- https://www.phpmyadmin.net/
--
-- Host: localhost
-- Generation Time: Apr 18, 2019 at 05:22 AM
-- Server version: 10.1.21-MariaDB
-- PHP Version: 7.1.1

SET SQL_MODE = "NO_AUTO_VALUE_ON_ZERO";
SET time_zone = "+00:00";


/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8mb4 */;

--
-- Database: `gophish`
--

-- --------------------------------------------------------

--
-- Table structure for table `bakery_user`
--

CREATE TABLE `bakery_user` (
  `uid` int(10) UNSIGNED NOT NULL COMMENT 'User ID on master site.',
  `master_uid` int(10) UNSIGNED NOT NULL COMMENT 'User ID on master site.'
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COMMENT='Keep track of UID on subsites, master only.';

--
-- Indexes for dumped tables
--

--
-- Indexes for table `bakery_user`
--
ALTER TABLE `bakery_user`
  ADD PRIMARY KEY (`uid`);

/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;


-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
