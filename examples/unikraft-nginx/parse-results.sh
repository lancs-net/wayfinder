#!/usr/bin/env bash

RESULTSDIR=$1
if [[ -z $RESULTSDIR ]]; then
  RESULTSDIR=$(pwd)/results
fi

TASKSFILE=${TASKS:-$RESULTSDIR/tasks.json}

if [[ ! -f $TASKSFILE ]]; then
  echo "Missing tasks.json file!"
  exit 1
fi

if [[ ! -d $RESULTSDIR ]]; then
  echo "Missing results director!"
  exit 1
fi

echo -n "TASKID,"
FIRST=y
for TASKID in $(cat $TASKSFILE | jq -r "keys[]"); do
  if [[ -f $RESULTSDIR/$TASKID/results.txt ]]; then
    cat $TASKSFILE | jq -r ".\"$TASKID\"|keys" | sed -e '1d' -e '$ d' | tr '\r\n' ' ' | sed 's/",   "/,/g' | sed 's/  "//g' | sed 's/" /,/g'
    cat $RESULTSDIR/$TASKID/results.txt | sed -e '6,10d' | wrkp | head -1
    break
  fi
done

for TASKID in $(cat $TASKSFILE | jq -r "keys[]"); do
  _jq() {
    cat $TASKSFILE | jq -r ".\"$TASKID\".\"${1}\""
  }

  echo -n "${TASKID},"

  cat $TASKSFILE | jq -r ".\"$TASKID\"" | gron | sed -e '1d' | awk -F'=' '{print $2}' | sed 's/["\ ;]//g' | tr ';\r\n' ','

  DAT=""
  if [[ -f $RESULTSDIR/$TASKID/results.txt ]]; then
    DAT=$(cat $RESULTSDIR/$TASKID/results.txt | sed -e '6,10d' | wrkp | sed -e '1,1d')
  fi
  if [[ $DAT != "" ]]; then
    echo $DAT
  else
    echo "X"
  fi
done

