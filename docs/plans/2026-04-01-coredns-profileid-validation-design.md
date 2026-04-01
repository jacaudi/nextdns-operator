# CoreDNS ProfileID Validation -- Design

**Issue:** #91
**Date:** 2026-04-01

## Problem

CoreDNS controller builds a Corefile without checking that ProfileID is non-empty. Race window exists where profile is Ready but ProfileID hasn't been set yet.

## Fix

Add empty ProfileID check after profile resolution (line 126), before Corefile generation. Set condition, requeue after 30s. Follows existing error handling pattern.
