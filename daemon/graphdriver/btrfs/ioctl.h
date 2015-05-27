/*
 copied from btrfs-progs/cmds-fi-usage.c HEAD of c061b303c08af7a58373908d0d77ef0f1161fd36
*/

/*
 * This program is free software; you can redistribute it and/or
 * modify it under the terms of the GNU General Public
 * License v2 as published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
 * General Public License for more details.
 *
 * You should have received a copy of the GNU General Public
 * License along with this program; if not, write to the
 * Free Software Foundation, Inc., 59 Temple Place - Suite 330,
 * Boston, MA 021110-1307, USA.
 */

#ifndef __BTRFS_DRIVER_IOCTL_H__
#define __BTRFS_DRIVER_IOCTL_H__

#include <stdio.h>
#include <stdlib.h>
#include <errno.h>
#include <string.h>
#include <btrfs/ioctl.h>
#include <btrfs/kerncompat.h>
#include <btrfs/ctree.h>


static int cmp_chunk_block_group(u64 f1, u64 f2)
{

	u64 mask;

	if ((f1 & BTRFS_BLOCK_GROUP_TYPE_MASK) ==
		(f2 & BTRFS_BLOCK_GROUP_TYPE_MASK))
			mask = BTRFS_BLOCK_GROUP_PROFILE_MASK;
	else if (f2 & BTRFS_BLOCK_GROUP_SYSTEM)
			return -1;
	else if (f1 & BTRFS_BLOCK_GROUP_SYSTEM)
			return +1;
	else
			mask = BTRFS_BLOCK_GROUP_TYPE_MASK;

	if ((f1 & mask) > (f2 & mask))
		return +1;
	else if ((f1 & mask) < (f2 & mask))
		return -1;
	else
		return 0;
}


static int cmp_btrfs_ioctl_space_info(const void *a, const void *b)
{
	return cmp_chunk_block_group(
		((struct btrfs_ioctl_space_info *)a)->flags,
		((struct btrfs_ioctl_space_info *)b)->flags);
}

static struct btrfs_ioctl_space_args *load_space_info(int fd, char *path)
{
	struct btrfs_ioctl_space_args *sargs = 0, *sargs_orig = 0;
	int e, ret, count;

	sargs_orig = sargs = calloc(1, sizeof(struct btrfs_ioctl_space_args));
	if (!sargs) {
		fprintf(stderr, "ERROR: not enough memory\n");
		return NULL;
	}

	sargs->space_slots = 0;
	sargs->total_spaces = 0;

	ret = ioctl(fd, BTRFS_IOC_SPACE_INFO, sargs);
	e = errno;
	if (ret) {
		fprintf(stderr,
			"ERROR: couldn't get space info on '%s' - %s\n",
			path, strerror(e));
		free(sargs);
		return NULL;
	}
	if (!sargs->total_spaces) {
		free(sargs);
		printf("No chunks found\n");
		return NULL;
	}

	count = sargs->total_spaces;

	sargs = realloc(sargs, sizeof(struct btrfs_ioctl_space_args) +
			(count * sizeof(struct btrfs_ioctl_space_info)));
	if (!sargs) {
		free(sargs_orig);
		fprintf(stderr, "ERROR: not enough memory\n");
		return NULL;
	}

	sargs->space_slots = count;
	sargs->total_spaces = 0;

	ret = ioctl(fd, BTRFS_IOC_SPACE_INFO, sargs);
	e = errno;

	if (ret) {
		fprintf(stderr,
			"ERROR: couldn't get space info on '%s' - %s\n",
			path, strerror(e));
		free(sargs);
		return NULL;
	}

	qsort(&(sargs->spaces), count, sizeof(struct btrfs_ioctl_space_info),
		cmp_btrfs_ioctl_space_info);

	return sargs;
}

#endif
